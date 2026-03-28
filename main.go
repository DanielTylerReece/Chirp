package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/tyler/gmessage/internal/app"
	"github.com/tyler/gmessage/internal/backend"
	"github.com/tyler/gmessage/internal/controller"
	"github.com/tyler/gmessage/internal/daemon"
	"github.com/tyler/gmessage/internal/db"
	"github.com/tyler/gmessage/internal/ui"
	"github.com/tyler/gmessage/internal/ui/content"
	"github.com/tyler/gmessage/internal/ui/newconversation"
	"github.com/tyler/gmessage/internal/ui/pairing"
	"github.com/tyler/gmessage/internal/ui/preferences"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--daemon" {
		runDaemon()
		return
	}

	cfg := app.NewConfig()

	application := adw.NewApplication("com.github.gmessage", gio.ApplicationFlagsNone)
	application.ConnectStartup(func() {
		ui.LoadCSS()
	})
	application.ConnectActivate(func() {
		win := ui.NewWindow(&application.Application, cfg)

		// Create app controller
		appCtrl, err := controller.NewApp(cfg)
		if err != nil {
			log.Printf("app init: %v", err)
			return
		}

		// Start event bus
		go appCtrl.Bus.Start()

		// Subscribe to real-time message events — refresh active conversation + sidebar
		appCtrl.Bus.SubscribeMessage(func(evt app.MessageEvent) {
			activeConv := win.ActiveConversationID()
			if activeConv != "" && activeConv == evt.ConversationID {
				msgs, err := appCtrl.DB.GetMessages(activeConv, 400, 0)
				if err == nil {
					glib.IdleAdd(func() {
						if win.ActiveConversationID() == activeConv {
							win.SetMessages(msgs)
						}
					})
				}
			}
			// Refresh sidebar to update last message preview
			convs, err := appCtrl.DB.ListConversations(100, 0)
			if err == nil {
				glib.IdleAdd(func() {
					win.UpdateConversations(convs)
				})
			}
		})

		// Wire logout
		win.SetOnLogout(func() {
			log.Println("Logging out — clearing session")
			if appCtrl.Client != nil {
				appCtrl.Client.Disconnect()
			}
			appCtrl.Session.Clear()
			application.Quit()
		})

		// Wire preferences
		win.SetOnShowPreferences(func() {
			pd := preferences.NewPreferencesDialog()
			pd.Present(win.ApplicationWindow())
		})

		// Wire new conversation button + Ctrl+N
		win.SetOnNewConversation(func() {
			dlg := newconversation.NewDialog()
			dlg.SetOnSearch(func(query string) []newconversation.ContactResult {
				// Search existing conversations by name
				convs, _ := appCtrl.DB.SearchConversations(query)
				var results []newconversation.ContactResult
				for _, c := range convs {
					results = append(results, newconversation.ContactResult{
						Name:           c.Name,
						PhoneNumber:    c.Name, // used as fallback display
						ConversationID: c.ID,
					})
				}
				return results
			})
			dlg.SetOnCreate(func(contacts []newconversation.ContactResult) {
				go func() {
					// If selecting an existing conversation, open it directly
					if len(contacts) == 1 && contacts[0].ConversationID != "" {
						convID := contacts[0].ConversationID
						conv, err := appCtrl.DB.GetConversation(convID)
						if err == nil && conv != nil {
							glib.IdleAdd(func() {
								win.SetConversation(conv)
								win.SelectConversation(convID)
							})
							return
						}
					}

					if appCtrl.Client == nil {
						log.Println("new conversation: not connected")
						return
					}
					var numbers []string
					for _, c := range contacts {
						if c.ConversationID == "" {
							numbers = append(numbers, c.PhoneNumber)
						}
					}
					if len(numbers) == 0 {
						return
					}
					convID, err := appCtrl.Client.GetOrCreateConversation(numbers)
					if err != nil {
						log.Printf("new conversation: %v", err)
						return
					}
					log.Printf("Created/found conversation: %s", convID)

					// Refresh sidebar and open the conversation
					convs, err := appCtrl.DB.ListConversations(100, 0)
					if err == nil {
						glib.IdleAdd(func() {
							win.UpdateConversations(convs)
						})
					}

					// Load and display the conversation
					conv, err := appCtrl.DB.GetConversation(convID)
					if err != nil || conv == nil {
						// Conversation may not be in DB yet; try fetching from phone
						convList, listErr := appCtrl.Client.ListConversations(100)
						if listErr == nil {
							for _, cd := range convList {
								if cd.ID == convID {
									dbConv := &db.Conversation{
										ID:                cd.ID,
										Name:              cd.Name,
										IsGroup:           cd.IsGroup,
										LastMessageTS:     cd.LastMessageTS,
										IsRCS:             cd.IsRCS,
										DefaultOutgoingID: cd.DefaultOutgoingID,
									}
									appCtrl.DB.UpsertConversation(dbConv)
									conv = dbConv
									break
								}
							}
						}
						if conv == nil {
							log.Printf("new conversation: could not load %s", convID)
							return
						}
					}

					glib.IdleAdd(func() {
						// Refresh sidebar again with new conversation
						convs, _ := appCtrl.DB.ListConversations(100, 0)
						win.UpdateConversations(convs)
						win.SetConversation(conv)
						win.SelectConversation(convID)
					})
				}()
			})
			dlg.Show(win.ApplicationWindow())
		})

		// Cached SIM options (populated once after connection, reused on every conversation switch)
		var cachedSIMOptions []content.SIMOption
		var cachedSIMInfos []backend.SIMInfo

		// Track active QR dialog so we can close it on pair success
		var activeQRDialog *pairing.QRDialog

		// Wire pairing callback
		appCtrl.OnNeedsPairing = func() {
			glib.IdleAdd(func() {
				activeQRDialog = showPairingDialog(win, appCtrl)
			})
		}

		// Load cached conversations immediately on startup (before connection)
		cachedConvs, _ := appCtrl.DB.ListConversations(100, 0)
		if len(cachedConvs) > 0 {
			log.Printf("Loaded %d cached conversations from DB", len(cachedConvs))
			win.UpdateConversations(cachedConvs)
		}

		appCtrl.OnConnected = func() {
			glib.IdleAdd(func() {
				log.Println("Connected! Syncing in background...")
				if appCtrl.Sync != nil {
					go func() {
						if err := appCtrl.Sync.ShallowBackfill(); err != nil {
							log.Printf("backfill error: %v", err)
							return
						}
						// Refresh sidebar with updated data
						convs, err := appCtrl.DB.ListConversations(100, 0)
						if err != nil {
							log.Printf("load conversations: %v", err)
							return
						}
						glib.IdleAdd(func() {
							win.UpdateConversations(convs)
						})
						if appCtrl.OnConversationsLoaded != nil {
							appCtrl.OnConversationsLoaded(convs)
						}

						// Backfill messages for empty conversations
						if err := appCtrl.Sync.BackfillEmptyConversations(); err != nil {
							log.Printf("backfill empty conversations: %v", err)
						} else {
							// Refresh sidebar after backfill
							convs, err := appCtrl.DB.ListConversations(100, 0)
							if err == nil {
								glib.IdleAdd(func() {
									win.UpdateConversations(convs)
								})
							}
						}

						// Fetch participant avatars in background
						go func() {
							ids, err := appCtrl.DB.ListParticipantIDsWithoutAvatar()
							if err != nil {
								log.Printf("avatars: list participant IDs: %v", err)
								return
							}
							log.Printf("avatars: %d participants need thumbnails", len(ids))
							if len(ids) == 0 {
								return
							}
							if err := appCtrl.Contacts.FetchAndCacheAvatars(ids); err != nil {
								log.Printf("avatars: fetch failed: %v", err)
								return
							}
							log.Println("avatars: fetch complete")
							// Refresh sidebar so avatars appear
							convs, err := appCtrl.DB.ListConversations(100, 0)
							if err == nil {
								glib.IdleAdd(func() {
									win.UpdateConversations(convs)
								})
							}
						}()
					}()
				}
			})
		}

		appCtrl.OnPairSuccess = func() {
			glib.IdleAdd(func() {
				log.Println("Pairing successful! Closing QR dialog and reconnecting...")
				if activeQRDialog != nil {
					activeQRDialog.Close()
					activeQRDialog = nil
				}
				appCtrl.Start()
			})
		}

		appCtrl.OnFatalError = func(err error) {
			glib.IdleAdd(func() {
				log.Printf("Fatal error: %v", err)
			})
		}

		// Subscribe to conversation events from the bus to refresh sidebar
		appCtrl.Bus.SubscribeConversation(func(evt app.ConversationEvent) {
			// Reload conversation list from DB on any conversation event
			convs, err := appCtrl.DB.ListConversations(100, 0)
			if err != nil {
				log.Printf("refresh conversations: %v", err)
				return
			}
			glib.IdleAdd(func() {
				win.UpdateConversations(convs)
			})
		})

		// Wire conversation selection → load messages
		win.SetOnConversationSelected(func(convID string) {
			// Get conversation from DB
			conv, err := appCtrl.DB.GetConversation(convID)
			if err != nil || conv == nil {
				log.Printf("get conversation %s: %v", convID, err)
				return
			}
			win.SetConversation(conv)

			// Update SIM selector using cache (fast)
			if len(cachedSIMOptions) >= 2 {
				var defaultSIM int32
				for _, s := range cachedSIMInfos {
					if s.ParticipantID == conv.DefaultOutgoingID {
						defaultSIM = s.SIMNumber
						break
					}
				}
				win.SetSIMs(cachedSIMOptions, defaultSIM)
			} else if appCtrl.Client != nil {
				// Populate cache on first use
				sims := appCtrl.Client.GetSIMs()
				cachedSIMInfos = sims
				cachedSIMOptions = nil
				for _, s := range sims {
					cachedSIMOptions = append(cachedSIMOptions, content.SIMOption{
						SIMNumber: s.SIMNumber,
						Label:     s.DisplayLabel(),
					})
				}
				if len(cachedSIMOptions) >= 2 {
					var defaultSIM int32
					for _, s := range sims {
						if s.ParticipantID == conv.DefaultOutgoingID {
							defaultSIM = s.SIMNumber
							break
						}
					}
					win.SetSIMs(cachedSIMOptions, defaultSIM)
				}
			}

			// Show cached messages from DB instantly
			cachedMsgs, _ := appCtrl.DB.GetMessages(convID, 400, 0)
			if len(cachedMsgs) > 0 {
				win.SetMessages(cachedMsgs)
			}

			// Then refresh from phone in background
			go func() {
				if appCtrl.Client == nil {
					return
				}
				msgs, err := appCtrl.Client.FetchMessages(convID, 400)
				if err != nil {
					log.Printf("fetch messages for %s: %v", convID, err)
					return
				}
				// Store in DB
				for _, m := range msgs {
					dbMsg := &db.Message{
						ID:              m.ID,
						ConversationID:  m.ConversationID,
						ParticipantID:   m.ParticipantID,
						Body:            m.Body,
						TimestampMS:     m.TimestampMS,
						IsFromMe:        m.IsFromMe,
						Status:          m.Status,
						MediaID:         m.MediaID,
						MediaMimeType:   m.MediaMimeType,
						MediaDecryptKey: m.MediaDecryptKey,
						MediaSize:       m.MediaSize,
						MediaWidth:      m.MediaWidth,
						MediaHeight:     m.MediaHeight,
						ThumbnailID:     m.ThumbnailID,
						ThumbnailKey:    m.ThumbnailKey,
					}
					appCtrl.DB.UpsertMessage(dbMsg)
				}
				// Refresh from DB only if user is still on this conversation
				if win.ActiveConversationID() == convID {
					dbMsgs, err := appCtrl.DB.GetMessages(convID, 400, 0)
					if err != nil {
						return
					}
					glib.IdleAdd(func() {
						if win.ActiveConversationID() == convID {
							win.SetMessages(dbMsgs)
						}
					})
				}
			}()
		})

		// Wire media loader for inline image display (cache-first)
		win.SetMediaLoader(func(mediaID string, decryptKey []byte) ([]byte, error) {
			// Check local disk cache first
			entry, _ := appCtrl.DB.GetMediaCache(mediaID)
			if entry != nil {
				data, err := os.ReadFile(entry.LocalPath)
				if err == nil {
					return data, nil
				}
				// File missing on disk — remove stale DB entry and re-download
				appCtrl.DB.DeleteMediaCache(mediaID)
			}

			if appCtrl.Client == nil {
				return nil, fmt.Errorf("not connected")
			}
			data, err := appCtrl.Client.DownloadMedia(mediaID, decryptKey)
			if err != nil {
				return nil, err
			}

			// Save to disk cache
			ext := ".bin"
			cachePath := filepath.Join(cfg.MediaDir, mediaID+ext)
			if writeErr := os.WriteFile(cachePath, data, 0600); writeErr == nil {
				appCtrl.DB.AddMediaCache(&db.MediaCacheEntry{
					MediaID:   mediaID,
					LocalPath: cachePath,
					MimeType:  "",
					CachedAt:  time.Now().Unix(),
					SizeBytes: int64(len(data)),
				})
			}

			return data, nil
		})

		// Wire send button
		win.SetOnSend(func(convID string, req content.SendRequest) {
			if appCtrl.Client == nil {
				log.Println("send: not connected")
				return
			}

			// Optimistic: insert a placeholder message immediately so the user sees it
			tmpID := fmt.Sprintf("tmp_%d", time.Now().UnixNano())
			body := req.Text
			if body == "" && len(req.MediaData) > 0 {
				body = "📷 Photo"
			}
			placeholder := &db.Message{
				ID:             tmpID,
				ConversationID: convID,
				Body:           body,
				TimestampMS:    time.Now().UnixMilli(),
				IsFromMe:       true,
				Status:         0, // hourglass
			}
			appCtrl.DB.UpsertMessage(placeholder)
			msgs, _ := appCtrl.DB.GetMessages(convID, 400, 0)
			glib.IdleAdd(func() {
				win.SetMessages(msgs)
			})

			go func() {
				var err error
				if len(req.MediaData) > 0 {
					err = appCtrl.Client.SendMediaMessage(convID, req.Text, req.MediaData, req.MediaName, req.MediaMime, req.SIMNumber)
				} else {
					err = appCtrl.Client.SendMessage(convID, req.Text, req.SIMNumber)
				}
				if err != nil {
					log.Printf("send message: %v", err)
					// Mark placeholder as failed
					appCtrl.DB.UpdateMessageStatus(tmpID, 4)
					dbMsgs, _ := appCtrl.DB.GetMessages(convID, 400, 0)
					glib.IdleAdd(func() { win.SetMessages(dbMsgs) })
					return
				}
				log.Printf("Message sent to %s", convID)

				// Remove placeholder — the re-fetch will insert the real message
				appCtrl.DB.DeleteMessage(tmpID)
				// Brief wait for server to process, then re-fetch from phone
				time.Sleep(500 * time.Millisecond)
				msgs, fetchErr := appCtrl.Client.FetchMessages(convID, 400)
				if fetchErr == nil {
					for _, m := range msgs {
						dbMsg := &db.Message{
							ID:              m.ID,
							ConversationID:  m.ConversationID,
							ParticipantID:   m.ParticipantID,
							Body:            m.Body,
							TimestampMS:     m.TimestampMS,
							IsFromMe:        m.IsFromMe,
							Status:          m.Status,
							MediaID:         m.MediaID,
							MediaMimeType:   m.MediaMimeType,
							MediaDecryptKey: m.MediaDecryptKey,
							MediaSize:       m.MediaSize,
							MediaWidth:      m.MediaWidth,
							MediaHeight:     m.MediaHeight,
							ThumbnailID:     m.ThumbnailID,
							ThumbnailKey:    m.ThumbnailKey,
						}
						appCtrl.DB.UpsertMessage(dbMsg)
					}
				} else {
					log.Printf("re-fetch messages after send: %v", fetchErr)
				}
				dbMsgs, err := appCtrl.DB.GetMessages(convID, 400, 0)
				if err == nil {
					glib.IdleAdd(func() {
						win.SetMessages(dbMsgs)
					})
				}
			}()
		})

		win.Present()

		// Start the app (check session, connect or prompt pairing)
		go appCtrl.Start()
	})
	os.Exit(application.Run(os.Args))
}

func showPairingDialog(win *ui.Window, appCtrl *controller.App) *pairing.QRDialog {
	qrDialog := pairing.NewQRDialog()

	go func() {
		url, err := appCtrl.StartPairing()
		if err != nil {
			log.Printf("pairing: %v", err)
			glib.IdleAdd(func() {
				qrDialog.SetStatus("Pairing failed: " + err.Error())
			})
			return
		}

		glib.IdleAdd(func() {
			qrDialog.SetQRCode(url)
			qrDialog.SetStatus("Scan this QR code with Google Messages on your phone\n\nGoogle Messages \u2192 Settings \u2192 Device pairing \u2192 QR code scanner")
		})
	}()

	qrDialog.Show(win.ApplicationWindow())
	return qrDialog
}

func runDaemon() {
	d, err := daemon.New()
	if err != nil {
		log.Fatalf("daemon: %v", err)
	}
	if err := d.Run(); err != nil {
		log.Fatalf("daemon: %v", err)
	}
}

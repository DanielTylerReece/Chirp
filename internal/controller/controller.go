package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/rs/zerolog"
	"github.com/tyler/gmessage/internal/app"
	"github.com/tyler/gmessage/internal/backend"
	"github.com/tyler/gmessage/internal/db"
	"go.mau.fi/mautrix-gmessages/pkg/libgm"
)

// App is the main application controller that wires backend, DB, and events.
type App struct {
	Config   *app.Config
	DB       *db.DB
	Bus      *app.EventBus
	Client   backend.GMClient
	Session  *backend.SessionManager
	Router   *backend.EventRouter
	Handler  *backend.MessageHandler
	Contacts *backend.ContactManager
	Sync     *backend.SyncEngine

	// Callbacks for UI
	OnPairSuccess        func()
	OnConnected          func()
	OnFatalError         func(error)
	OnNeedsPairing       func()
	OnConversationsLoaded func(convs []db.Conversation)
}

func NewApp(config *app.Config) (*App, error) {
	if err := config.EnsureDirs(); err != nil {
		return nil, err
	}

	database, err := db.Open(config.DBPath)
	if err != nil {
		return nil, err
	}

	bus := app.NewEventBus()
	session := backend.NewSessionManager(config.DataDir)
	handler := backend.NewMessageHandler(database, bus)

	a := &App{
		Config:  config,
		DB:      database,
		Bus:     bus,
		Session: session,
		Handler: handler,
	}

	return a, nil
}

// Start checks for existing session and either connects or signals need for pairing.
func (a *App) Start() {
	sessionData, err := a.Session.Load()
	if err != nil {
		log.Printf("app: load session: %v", err)
	}

	if sessionData == nil {
		log.Println("app: no session found, need pairing")
		if a.OnNeedsPairing != nil {
			a.OnNeedsPairing()
		}
		return
	}

	// Restore session
	var authData libgm.AuthData
	if err := json.Unmarshal(sessionData, &authData); err != nil {
		log.Printf("app: unmarshal session: %v", err)
		if a.OnNeedsPairing != nil {
			a.OnNeedsPairing()
		}
		return
	}

	a.connect(&authData)
}

// StartPairing initiates QR code pairing. Returns the QR URL.
func (a *App) StartPairing() (string, error) {
	logger := zerolog.New(log.Writer()).With().Timestamp().Logger()
	realClient := backend.NewRealClient(nil, logger)
	a.Client = realClient
	a.setupRouter()

	// Override pair success to save session
	a.Router.OnPairSuccess = func() {
		data, err := json.Marshal(realClient.AuthData())
		if err == nil {
			a.Session.Save(data)
		}
		log.Println("app: pairing successful, session saved")
		if a.OnPairSuccess != nil {
			a.OnPairSuccess()
		}
	}

	realClient.SetEventHandler(a.Router.Handle)

	log.Println("controller: calling StartLogin...")
	url, err := realClient.StartLogin()
	if err != nil {
		log.Printf("controller: StartLogin error: %v", err)
		return "", err
	}
	log.Printf("controller: got QR URL (%d chars): %s", len(url), url[:min(80, len(url))]+"...")
	return url, nil
}

// StartGoogleLogin initiates Google Account (Gaia) pairing with browser cookies.
// Returns channels for emoji display and errors.
func (a *App) StartGoogleLogin(cookies map[string]string) (emojiChan chan string, errChan chan error) {
	emojiChan = make(chan string, 1)
	errChan = make(chan error, 1)

	// Disconnect any existing client (e.g., from QR flow)
	if a.Client != nil {
		a.Client.Disconnect()
	}

	logger := zerolog.New(log.Writer()).With().Timestamp().Logger()
	realClient := backend.NewRealClientWithCookies(cookies, logger)
	a.Client = realClient
	a.setupRouter()

	realClient.SetEventHandler(a.Router.Handle)

	go func() {
		log.Println("controller: fetching Google config...")
		if err := realClient.FetchConfig(context.Background()); err != nil {
			log.Printf("controller: FetchConfig error: %v", err)
			errChan <- fmt.Errorf("cookies expired or invalid. Make sure you:\n1. Open messages.google.com in your browser\n2. Sign in to your Google account\n3. Re-copy the cookies (they refresh on each visit)")
			return
		}
		log.Println("controller: starting Google Account pairing...")
		err := realClient.DoGaiaPairing(context.Background(), func(emoji string) {
			log.Printf("controller: emoji for verification: %s", emoji)
			emojiChan <- emoji
		})
		if err != nil {
			log.Printf("controller: Gaia pairing error: %v", err)
			// Provide a friendlier error message for auth failures
			errMsg := err.Error()
			if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "invalid authentication") {
				err = fmt.Errorf("cookies expired or invalid. Make sure you:\n1. Open messages.google.com in your browser\n2. Sign in to your Google account\n3. Re-copy the cookies (they refresh on each visit)")
			}
			errChan <- err
			return
		}

		// Save session
		data, err := json.Marshal(realClient.AuthData())
		if err == nil {
			a.Session.Save(data)
		}
		log.Println("controller: Google Account pairing successful, session saved")

		if a.OnPairSuccess != nil {
			a.OnPairSuccess()
		}
	}()

	return emojiChan, errChan
}

func (a *App) connect(authData *libgm.AuthData) {
	logger := zerolog.New(log.Writer()).With().Timestamp().Logger()
	realClient := backend.NewRealClient(authData, logger)
	a.Client = realClient
	a.setupRouter()

	realClient.SetEventHandler(a.Router.Handle)

	go func() {
		if err := realClient.Connect(); err != nil {
			log.Printf("app: connect: %v", err)
			if a.OnFatalError != nil {
				a.OnFatalError(err)
			}
			return
		}
		log.Println("app: connected")
		if a.OnConnected != nil {
			a.OnConnected()
		}
	}()
}

func (a *App) setupRouter() {
	a.Router = backend.NewEventRouter(a.Bus)
	a.Contacts = backend.NewContactManager(a.Client, a.DB, a.Config)
	a.Sync = backend.NewSyncEngine(a.Client, a.DB, a.Bus, a.Config)

	a.Router.OnAuthRefreshed = func() {
		if rc, ok := a.Client.(*backend.RealClient); ok {
			data, err := json.Marshal(rc.AuthData())
			if err == nil {
				a.Session.Save(data)
			}
		}
	}

	a.Router.OnFatalError = func(err error) {
		if a.OnFatalError != nil {
			a.OnFatalError(err)
		}
	}

	// Persist incoming/updated messages to DB in real-time
	a.Router.OnMessage = func(wrapped *libgm.WrappedMessage) {
		m := wrapped.Message
		if m == nil {
			return
		}
		msgID := m.GetMessageID()
		convID := m.GetConversationID()
		if msgID == "" || convID == "" {
			return
		}

		// Try to load existing message to merge with
		existing, _ := a.DB.GetMessage(msgID)

		// Start from existing or blank
		var dbMsg *db.Message
		if existing != nil {
			dbMsg = existing
		} else {
			dbMsg = &db.Message{
				ID:             msgID,
				ConversationID: convID,
			}
		}

		// Always update participant and timestamp if present
		if pid := m.GetParticipantID(); pid != "" {
			dbMsg.ParticipantID = pid
		}
		if ts := m.GetTimestamp(); ts > 0 {
			dbMsg.TimestampMS = ts / 1000 // libgm returns microseconds
		}

		// Update status
		if ms := m.GetMessageStatus(); ms != nil {
			statusVal := int(ms.GetStatus())
			dbMsg.IsFromMe = statusVal > 0 && statusVal < 100
			dbMsg.Status = backend.ConvertMessageStatus(ms.GetStatus())
		}

		// Update content only if present (don't overwrite with empty)
		for _, info := range m.GetMessageInfo() {
			if mc := info.GetMessageContent(); mc != nil {
				if content := mc.GetContent(); content != "" {
					dbMsg.Body = content
				}
			}
			if media := info.GetMediaContent(); media != nil {
				if mid := media.GetMediaID(); mid != "" {
					dbMsg.MediaID = mid
				}
				if mime := media.GetMimeType(); mime != "" {
					dbMsg.MediaMimeType = mime
				}
				if key := media.GetDecryptionKey(); len(key) > 0 {
					dbMsg.MediaDecryptKey = key
				}
				if sz := media.GetSize(); sz > 0 {
					dbMsg.MediaSize = sz
				}
				if dims := media.GetDimensions(); dims != nil {
					if w := int(dims.GetWidth()); w > 0 {
						dbMsg.MediaWidth = w
					}
					if h := int(dims.GetHeight()); h > 0 {
						dbMsg.MediaHeight = h
					}
				}
				if tid := media.GetThumbnailMediaID(); tid != "" {
					dbMsg.ThumbnailID = tid
				}
				if tkey := media.GetThumbnailDecryptionKey(); len(tkey) > 0 {
					dbMsg.ThumbnailKey = tkey
				}
			}
		}

		a.DB.UpsertMessage(dbMsg)

		// Update conversation preview
		preview := dbMsg.Body
		if preview == "" && dbMsg.MediaID != "" {
			preview = "📷 Photo"
		}
		if preview != "" {
			a.DB.UpdateConversationPreview(convID, preview, dbMsg.TimestampMS)
		}
	}
}

// Close shuts down the app.
func (a *App) Close() {
	if a.Client != nil {
		a.Client.Disconnect()
	}
	a.Bus.Stop()
	a.DB.Close()
}

package controller

import (
	"encoding/json"
	"log"

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

		dbMsg := &db.Message{
			ID:             msgID,
			ConversationID: convID,
			ParticipantID:  m.GetParticipantID(),
			TimestampMS:    m.GetTimestamp() / 1000, // libgm returns microseconds
		}

		// Status
		if ms := m.GetMessageStatus(); ms != nil {
			statusVal := int(ms.GetStatus())
			dbMsg.IsFromMe = statusVal > 0 && statusVal < 100
			dbMsg.Status = backend.ConvertMessageStatus(ms.GetStatus())
		}

		// Text and media from MessageInfo
		for _, info := range m.GetMessageInfo() {
			if mc := info.GetMessageContent(); mc != nil {
				dbMsg.Body = mc.GetContent()
			}
			if media := info.GetMediaContent(); media != nil && dbMsg.MediaID == "" {
				dbMsg.MediaID = media.GetMediaID()
				dbMsg.MediaMimeType = media.GetMimeType()
				dbMsg.MediaDecryptKey = media.GetDecryptionKey()
				dbMsg.MediaSize = media.GetSize()
				if dims := media.GetDimensions(); dims != nil {
					dbMsg.MediaWidth = int(dims.GetWidth())
					dbMsg.MediaHeight = int(dims.GetHeight())
				}
				dbMsg.ThumbnailID = media.GetThumbnailMediaID()
				dbMsg.ThumbnailKey = media.GetThumbnailDecryptionKey()
			}
		}

		a.DB.UpsertMessage(dbMsg)
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

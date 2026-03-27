package daemon

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/tyler/gmessage/internal/app"
	"github.com/tyler/gmessage/internal/backend"
	"github.com/tyler/gmessage/internal/db"
	"go.mau.fi/mautrix-gmessages/pkg/libgm"
)

// Daemon runs GMessage in headless mode — libgm + SQLite + notifications.
type Daemon struct {
	config  *app.Config
	db      *db.DB
	client  *backend.RealClient
	bus     *app.EventBus
	router  *backend.EventRouter
	handler *backend.MessageHandler
	session *backend.SessionManager
}

func New() (*Daemon, error) {
	config := app.NewConfig()
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

	return &Daemon{
		config:  config,
		db:      database,
		bus:     bus,
		handler: handler,
		session: session,
	}, nil
}

// Run starts the daemon and blocks until shutdown signal.
func (d *Daemon) Run() error {
	// Load session
	sessionData, err := d.session.Load()
	if err != nil {
		return err
	}
	if sessionData == nil {
		log.Println("daemon: no session found, run gmessage to pair first")
		return nil
	}

	var authData libgm.AuthData
	if err := json.Unmarshal(sessionData, &authData); err != nil {
		return err
	}

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	d.client = backend.NewRealClient(&authData, logger)

	// Set up event routing
	d.router = backend.NewEventRouter(d.bus)
	d.router.OnAuthRefreshed = func() {
		// Persist refreshed token
		data, err := json.Marshal(d.client.AuthData())
		if err == nil {
			d.session.Save(data)
		}
	}
	d.client.SetEventHandler(d.router.Handle)

	// Connect
	log.Println("daemon: connecting...")
	if err := d.client.Connect(); err != nil {
		return err
	}
	log.Println("daemon: connected")

	// Start event bus
	go d.bus.Start()

	// Wait for shutdown signal
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	log.Println("daemon: shutting down...")

	d.client.Disconnect()
	d.bus.Stop()
	d.db.Close()

	return nil
}

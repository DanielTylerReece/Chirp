package main

import (
	"log"
	"os"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/tyler/gmessage/internal/app"
	"github.com/tyler/gmessage/internal/daemon"
	"github.com/tyler/gmessage/internal/ui"
	"github.com/tyler/gmessage/internal/ui/preferences"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--daemon" {
		runDaemon()
		return
	}

	cfg := app.NewConfig()
	if err := cfg.EnsureDirs(); err != nil {
		log.Fatalf("gmessage: failed to create directories: %v", err)
	}

	application := adw.NewApplication("com.github.gmessage", gio.ApplicationFlagsNone)
	application.ConnectStartup(func() {
		ui.LoadCSS()
	})
	application.ConnectActivate(func() {
		win := ui.NewWindow(&application.Application, cfg)

		// Wire preferences dialog with app-level state
		win.SetOnShowPreferences(func() {
			pd := preferences.NewPreferencesDialog()
			pd.Present(win.ApplicationWindow())
		})

		win.Present()
	})
	os.Exit(application.Run(os.Args))
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

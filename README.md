<p align="center">
  <img src="data/com.github.gmessage.svg" width="128" height="128" alt="Chirp icon"/>
</p>

<h1 align="center">Chirp</h1>

<p align="center">
  <strong>A native Google Messages client for GNOME</strong>
</p>

<p align="center">
  Pair your Android phone and send/receive SMS & RCS messages right from your Linux desktop — no browser, no Electron, just a clean GTK4 app that feels at home on GNOME.
</p>

---

## Features

- **Native GNOME experience** — GTK4 + libadwaita, follows system theme and accent colors
- **iMessage-style chat bubbles** — sent/received alignment, delivery status indicators, timestamps
- **QR code pairing** — scan once from Google Messages on your phone, just like the web client
- **Inline media** — send and receive images with click-to-expand fullscreen viewer
- **Dual-SIM support** — pick which SIM to send from per-conversation
- **Instant startup** — local SQLite cache loads conversations immediately, syncs in background
- **Real-time delivery** — messages appear as they arrive, status updates from sending to delivered
- **New conversations** — start chats with contact search and phone number entry
- **Desktop notifications** — background daemon keeps you connected when the window is closed
- **Session stored in GNOME Keyring** — credentials never touch plaintext on disk
- **Offline access** — read your full message history without a connection

## Screenshots

> *Coming soon*

## Requirements

| Dependency | Version |
|---|---|
| GTK4 | 4.12+ |
| libadwaita | 1.4+ |
| Go | 1.22+ (build only) |
| Android phone | Google Messages app |

## Build & Install

### Fedora / RHEL

```bash
sudo dnf install gtk4-devel libadwaita-devel gcc golang
git clone https://github.com/DanielTylerReece/Chirp.git
cd Chirp
make install
```

### Arch / CachyOS

```bash
sudo pacman -S gtk4 libadwaita go gcc
git clone https://github.com/DanielTylerReece/Chirp.git
cd Chirp
make install
```

### Ubuntu / Debian

```bash
sudo apt install libgtk-4-dev libadwaita-1-dev gcc golang-go
git clone https://github.com/DanielTylerReece/Chirp.git
cd Chirp
make install
```

`make install` builds the binary and installs it to `~/.local/bin/`, along with the desktop entry, icon, and GSettings schema to the appropriate XDG directories.

To build without installing:

```bash
make build
./gmessage
```

## Usage

1. Launch **Chirp** from your app grid or run `gmessage`
2. Scan the QR code with your phone:
   **Google Messages** > **Settings** > **Device pairing** > **QR code scanner**
3. Start messaging

### Background notifications

```bash
systemctl --user enable --now gmessage-daemon
```

### Log out

Use the **three-dot menu** in the top-left corner > **Log out** to unpair and clear your session.

## How it works

Chirp uses [libgm](https://github.com/mautrix/gmessages) — the same protocol library that powers the [mautrix-gmessages](https://github.com/mautrix/gmessages) Matrix bridge. It communicates with Google's servers through a long-polling relay, acting as an additional paired device alongside your phone.

Your phone must remain on and connected to the internet for messages to flow. This is the same model as Google Messages for Web.

## Project structure

```
main.go                      App entry point and wiring
internal/
  app/                       Config, event bus
  backend/                   libgm client wrapper, session, contacts, sync
  controller/                App lifecycle, pairing flow
  daemon/                    Background notification service
  db/                        SQLite schema, migrations, CRUD
  ui/
    sidebar/                 Conversation list
    content/                 Message display, compose bar, media viewer
    pairing/                 QR code dialog
    newconversation/         New chat dialog
    preferences/             Settings dialog
style/style.css              GTK4 stylesheet
data/                        Desktop entry, icon, systemd unit
```

## License

AGPL-3.0-or-later (required by the libgm dependency)

## Acknowledgments

- [mautrix/gmessages](https://github.com/mautrix/gmessages) — the libgm protocol library that makes this possible
- [gotk4](https://github.com/diamondburned/gotk4) — Go bindings for GTK4
- [gotk4-adwaita](https://github.com/diamondburned/gotk4-adwaita) — Go bindings for libadwaita

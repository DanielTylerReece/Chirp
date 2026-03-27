# GMessage

Native GTK4/libadwaita Google Messages client for GNOME. Pairs with your Android phone to send and receive SMS/RCS messages from your desktop.

Built on [libgm](https://github.com/mautrix/gmessages) (the mautrix Google Messages protocol library).

## Features

- Native GNOME desktop app (GTK4 + libadwaita)
- iMessage-style chat bubbles
- Contact photos from Google contacts
- Full-text message search
- Desktop notifications
- Media sending (photos, files)
- Reply threading and reactions
- Typing indicators and read receipts
- Background daemon for always-on notifications
- Session stored in GNOME Keyring
- Local SQLite database with offline access

## Requirements

- GTK4 4.12+
- libadwaita 1.4+
- Go 1.25+ (build only)
- Android phone with Google Messages

## Build

```bash
go build -o gmessage .
```

## Install

```bash
make install
```

Or on Arch Linux:
```bash
cd packaging/aur && makepkg -si
```

## Usage

1. Run `gmessage`
2. Pair with your phone (scan QR code from Google Messages → Settings → Device pairing)
3. Start messaging

For background notifications:
```bash
systemctl --user enable --now gmessage-daemon
```

## License

AGPL-3.0-or-later (required by libgm dependency)

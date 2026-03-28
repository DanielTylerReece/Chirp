#!/bin/bash
set -e

PREFIX="${PREFIX:-$HOME/.local}"
APP_ID="com.github.gmessage"

echo "Installing Chirp to ${PREFIX}..."

install -Dm755 chirp "${PREFIX}/bin/chirp"
install -Dm644 "data/${APP_ID}.desktop" "${PREFIX}/share/applications/${APP_ID}.desktop"
install -Dm644 "data/${APP_ID}.svg" "${PREFIX}/share/icons/hicolor/scalable/apps/${APP_ID}.svg"
install -Dm644 "data/${APP_ID}.gschema.xml" "${PREFIX}/share/glib-2.0/schemas/${APP_ID}.gschema.xml"
install -Dm644 "data/${APP_ID}.metainfo.xml" "${PREFIX}/share/metainfo/${APP_ID}.metainfo.xml"
install -Dm644 "data/gmessage-daemon.service" "${HOME}/.config/systemd/user/gmessage-daemon.service"
install -Dm644 "data/${APP_ID}.Daemon.desktop" "${HOME}/.config/autostart/${APP_ID}.Daemon.desktop"

# Compile GSettings schemas
glib-compile-schemas "${PREFIX}/share/glib-2.0/schemas/" 2>/dev/null || true

# Update icon cache
gtk-update-icon-cache -f "${PREFIX}/share/icons/hicolor/" 2>/dev/null || true

# Update desktop database
update-desktop-database "${PREFIX}/share/applications/" 2>/dev/null || true

echo "Chirp installed successfully!"
echo ""
echo "Run 'chirp' to start."
echo "Run 'chirp --daemon' to start the background service."
echo "Run 'systemctl --user enable --now gmessage-daemon' for always-on notifications."

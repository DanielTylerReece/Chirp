APP_ID   := com.github.gmessage
BINARY   := gmessage
PREFIX   := $(HOME)/.local

.PHONY: build test vet gate install run clean

build:
	go build -o $(BINARY) .

test:
	go test ./... -v -count=1 -timeout 60s

test-race:
	go test ./... -race -count=1 -timeout 120s

vet:
	go vet ./...

gate: build test vet test-race
	@echo "Phase gate passed."

install: build
	install -Dm755 $(BINARY) $(PREFIX)/bin/$(BINARY)
	install -Dm644 data/$(APP_ID).desktop $(PREFIX)/share/applications/$(APP_ID).desktop
	install -Dm644 data/$(APP_ID).svg $(PREFIX)/share/icons/hicolor/scalable/apps/$(APP_ID).svg
	install -Dm644 data/$(APP_ID).gschema.xml $(PREFIX)/share/glib-2.0/schemas/$(APP_ID).gschema.xml
	glib-compile-schemas $(PREFIX)/share/glib-2.0/schemas/
	install -Dm644 data/gmessage-daemon.service $(HOME)/.config/systemd/user/gmessage-daemon.service
	install -Dm644 data/$(APP_ID).Daemon.desktop $(HOME)/.config/autostart/$(APP_ID).Daemon.desktop

run: build
	GMESSAGE_LOG_LEVEL=debug ./$(BINARY)

clean:
	rm -f $(BINARY)
	go clean -testcache

BIN      = winapps-systray
BUILDDIR = build
GO       = $(shell which go 2>/dev/null || echo "/usr/local/go/bin/go")

PREFIX  ?= /usr/local
BINDIR   = $(PREFIX)/bin
DATADIR  = $(PREFIX)/share

help:
	@echo "Targets:"
	@echo "  build              Compile the binary to build/"
	@echo "  install            Install system-wide (requires root)"
	@echo "  install-autostart  Install + add to /etc/xdg/autostart"
	@echo "  user-install       Install to ~/.local and add to autostart"
	@echo "  user-uninstall     Remove user installation"
	@echo "  uninstall          Remove system-wide installation"
	@echo "  rpm                Build an RPM package (requires nfpm)"
	@echo "  deb                Build a DEB package (requires nfpm)"
	@echo "  clean              Remove build/"

all: build

build:
	@mkdir -p $(BUILDDIR)
	$(GO) build -o $(BUILDDIR)/$(BIN) main.go

install: build
	install -Dm755 $(BUILDDIR)/$(BIN)                  $(DESTDIR)$(BINDIR)/$(BIN)
	install -Dm644 dist/winapps-systray.desktop        $(DESTDIR)$(DATADIR)/applications/winapps-systray.desktop
	install -Dm644 dist/winapps-systray.svg            $(DESTDIR)$(DATADIR)/icons/hicolor/scalable/apps/winapps-systray.svg

install-autostart: install
	install -Dm644 dist/winapps-systray-autostart.desktop $(DESTDIR)/etc/xdg/autostart/winapps-systray.desktop

user-install: build
	install -Dm755 $(BUILDDIR)/$(BIN) $(HOME)/.local/bin/$(BIN)
	install -Dm644 dist/winapps-systray.desktop $(HOME)/.local/share/applications/winapps-systray.desktop
	install -Dm644 dist/winapps-systray.svg $(HOME)/.local/share/icons/hicolor/scalable/apps/winapps-systray.svg
	mkdir -p $(HOME)/.config/autostart
	install -Dm644 dist/winapps-systray-autostart.desktop $(HOME)/.config/autostart/winapps-systray.desktop

user-uninstall:
	rm -f $(HOME)/.local/bin/$(BIN)
	rm -f $(HOME)/.local/share/applications/winapps-systray.desktop
	rm -f $(HOME)/.local/share/icons/hicolor/scalable/apps/winapps-systray.svg
	rm -f $(HOME)/.config/autostart/winapps-systray.desktop

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BIN)
	rm -f $(DESTDIR)$(DATADIR)/applications/winapps-systray.desktop
	rm -f $(DESTDIR)$(DATADIR)/icons/hicolor/scalable/apps/winapps-systray.svg
	rm -f $(DESTDIR)/etc/xdg/autostart/winapps-systray.desktop

rpm: build
	nfpm package --packager rpm --target $(BUILDDIR)/

deb: build
	nfpm package --packager deb --target $(BUILDDIR)/

clean:
	rm -rf $(BUILDDIR)

.PHONY: all help build install install-autostart user-install user-uninstall uninstall rpm deb clean

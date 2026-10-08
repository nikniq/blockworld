# Packaging targets (included from the Makefile). Each must run on its platform.

APP_NAME := Blockworld
BUNDLE   := dist/$(APP_NAME).app

icon.png: cmd/mkicon/main.go
	go run ./cmd/mkicon

# macOS: a double-clickable .app with an .icns icon. Unsigned; first launch may
# need right click > Open, or run: xattr -dr com.apple.quarantine dist/Blockworld.app
app-macos: icon.png
	go build -ldflags="-s -w" -o blockworld .
	rm -rf $(BUNDLE)
	mkdir -p $(BUNDLE)/Contents/MacOS $(BUNDLE)/Contents/Resources dist/icon.iconset
	for s in 16 32 64 128 256 512; do \
	  sips -z $$s $$s icon.png --out dist/icon.iconset/icon_$${s}x$${s}.png >/dev/null; \
	  d=$$((s*2)); sips -z $$d $$d icon.png --out dist/icon.iconset/icon_$${s}x$${s}@2x.png >/dev/null; \
	done
	iconutil -c icns dist/icon.iconset -o $(BUNDLE)/Contents/Resources/blockworld.icns
	rm -rf dist/icon.iconset
	cp blockworld $(BUNDLE)/Contents/MacOS/blockworld
	printf '%s\n' \
	  '<?xml version="1.0" encoding="UTF-8"?>' \
	  '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' \
	  '<plist version="1.0"><dict>' \
	  '  <key>CFBundleName</key><string>Blockworld</string>' \
	  '  <key>CFBundleDisplayName</key><string>Blockworld</string>' \
	  '  <key>CFBundleIdentifier</key><string>com.blockworld.game</string>' \
	  '  <key>CFBundleVersion</key><string>1.0</string>' \
	  '  <key>CFBundleShortVersionString</key><string>1.0</string>' \
	  '  <key>CFBundleExecutable</key><string>blockworld</string>' \
	  '  <key>CFBundleIconFile</key><string>blockworld</string>' \
	  '  <key>CFBundlePackageType</key><string>APPL</string>' \
	  '  <key>NSHighResolutionCapable</key><true/>' \
	  '  <key>LSMinimumSystemVersion</key><string>11.0</string>' \
	  '</dict></plist>' > $(BUNDLE)/Contents/Info.plist
	@echo "Built $(BUNDLE)"

# Linux: install the binary, icon and a desktop launcher for the current user.
app-linux: icon.png
	go build -ldflags="-s -w" -o blockworld .
	mkdir -p ~/.local/bin ~/.local/share/applications ~/.local/share/icons/hicolor/512x512/apps
	cp blockworld ~/.local/bin/blockworld
	cp icon.png ~/.local/share/icons/hicolor/512x512/apps/blockworld.png
	printf '%s\n' \
	  '[Desktop Entry]' 'Type=Application' 'Name=Blockworld' \
	  'Comment=Mine by day, survive the night' \
	  'Exec=$(HOME)/.local/bin/blockworld' 'Icon=blockworld' \
	  'Terminal=false' 'Categories=Game;' > ~/.local/share/applications/blockworld.desktop
	-update-desktop-database ~/.local/share/applications 2>/dev/null
	-gtk-update-icon-cache ~/.local/share/icons/hicolor 2>/dev/null
	@echo "Installed: find Blockworld in your application menu"

# Windows: embed the icon into the .exe with go-winres (go install github.com/tc-hib/go-winres@latest).
app-windows: icon.png
	go-winres simply --icon icon.png --product-name Blockworld --file-description "Blockworld" --product-version 1.0 --file-version 1.0
	go build -ldflags="-s -w -H windowsgui" -o blockworld.exe .
	@echo "Built blockworld.exe with an embedded icon"

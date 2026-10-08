#!/usr/bin/env bash
# cx.sh - build and package Blockworld for the platform this script runs on.
#
#   ./cx.sh                 test, build and package into dist/
#   ./cx.sh --version 1.2   set the version stamped into the package names and metadata
#   ./cx.sh --skip-tests    package without running the test suite
#   ./cx.sh --run           launch the packaged game afterwards
#   ./cx.sh --clean         remove dist/ and build outputs
#
# Output (always under dist/):
#   macOS    Blockworld.app, Blockworld-<ver>-macos-<arch>.zip
#   Linux    blockworld-<ver>-linux-<arch>.tar.gz (binary, icon, .desktop, install.sh)
#   Windows  blockworld-<ver>-windows-<arch>.zip (blockworld.exe with embedded icon)
# plus SHA256SUMS. raylib is compiled through cgo, so each platform packages itself.
set -euo pipefail

cd "$(dirname "$0")"

VERSION="1.0"
RUN_TESTS=1
LAUNCH=0
for arg in "$@"; do
  case "$arg" in
    --version) ;;                       # value handled below
    --version=*) VERSION="${arg#*=}" ;;
    --skip-tests) RUN_TESTS=0 ;;
    --run) LAUNCH=1 ;;
    --clean) rm -rf dist blockworld blockworld.exe ./*.syso; echo "cleaned"; exit 0 ;;
    -h|--help) sed -n '2,16p' "$0"; exit 0 ;;
  esac
done
# "--version 1.2" form
prev=""
for arg in "$@"; do
  if [ "$prev" = "--version" ]; then VERSION="$arg"; fi
  prev="$arg"
done

OS="$(uname -s)"
ARCH="$(uname -m)"
case "$ARCH" in x86_64|amd64) ARCH=x64 ;; arm64|aarch64) ARCH=arm64 ;; esac

say()  { printf '\033[1;36m==>\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

need() {
  command -v "$1" >/dev/null 2>&1 || fail "$1 is required: $2"
}

# ---------- toolchain checks ----------
need go "install Go 1.22 or newer from https://go.dev/dl/"
case "$OS" in
  Darwin)
    need clang "run: xcode-select --install"
    need iconutil "ships with macOS"
    need sips "ships with macOS"
    ;;
  Linux)
    need gcc "e.g. sudo apt install build-essential, or sudo dnf install gcc"
    for h in GL/gl.h X11/Xlib.h; do
      if ! echo "#include <$h>" | gcc -E - >/dev/null 2>&1; then
        fail "missing development header $h. Debian/Ubuntu: sudo apt install libgl1-mesa-dev libx11-dev libxi-dev libxcursor-dev libxrandr-dev libxinerama-dev libwayland-dev libxkbcommon-dev   Fedora: sudo dnf install mesa-libGL-devel libX11-devel libXi-devel libXcursor-devel libXrandr-devel libXinerama-devel wayland-devel libxkbcommon-devel"
      fi
    done
    ;;
  MINGW*|MSYS*|CYGWIN*|Windows_NT)
    OS=Windows
    need gcc "install a GCC toolchain such as w64devkit or MSYS2 mingw-w64 and put gcc on PATH"
    if ! command -v go-winres >/dev/null 2>&1; then
      say "installing go-winres (embeds the icon into the .exe)"
      go install github.com/tc-hib/go-winres@latest
      export PATH="$PATH:$(go env GOPATH)/bin"
      need go-winres "go install github.com/tc-hib/go-winres@latest and add \$(go env GOPATH)/bin to PATH"
    fi
    ;;
  *) fail "unsupported platform: $OS" ;;
esac

mkdir -p dist
say "Blockworld $VERSION for $OS/$ARCH"

# ---------- icon, tests, build ----------
say "generating icon"
go run ./cmd/mkicon

if [ "$RUN_TESTS" = 1 ]; then
  say "running tests"
  go test ./... >/dev/null || { go test ./... 2>&1 | tail -20; fail "tests failed"; }
fi

LDFLAGS="-s -w -X main.gameVersion=$VERSION"

# ---------- package ----------
case "$OS" in
  Darwin)
    say "building"
    go build -ldflags="$LDFLAGS" -o blockworld .
    APP="dist/Blockworld.app"
    say "assembling $APP"
    rm -rf "$APP" dist/icon.iconset
    mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" dist/icon.iconset
    for s in 16 32 64 128 256 512; do
      sips -z "$s" "$s" icon.png --out "dist/icon.iconset/icon_${s}x${s}.png" >/dev/null
      d=$((s*2)); sips -z "$d" "$d" icon.png --out "dist/icon.iconset/icon_${s}x${s}@2x.png" >/dev/null
    done
    iconutil -c icns dist/icon.iconset -o "$APP/Contents/Resources/blockworld.icns"
    rm -rf dist/icon.iconset
    cp blockworld "$APP/Contents/MacOS/blockworld"
    cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>CFBundleName</key><string>Blockworld</string>
  <key>CFBundleDisplayName</key><string>Blockworld</string>
  <key>CFBundleIdentifier</key><string>com.blockworld.game</string>
  <key>CFBundleVersion</key><string>$VERSION</string>
  <key>CFBundleShortVersionString</key><string>$VERSION</string>
  <key>CFBundleExecutable</key><string>blockworld</string>
  <key>CFBundleIconFile</key><string>blockworld</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>NSHighResolutionCapable</key><true/>
  <key>LSMinimumSystemVersion</key><string>11.0</string>
</dict></plist>
PLIST
    say "signing (ad hoc)"
    codesign --force --deep --sign - "$APP"
    ZIP="dist/Blockworld-$VERSION-macos-$ARCH.zip"
    rm -f "$ZIP"
    (cd dist && ditto -c -k --keepParent Blockworld.app "$(basename "$ZIP")")
    OUT="$APP"
    ;;
  Linux)
    say "building"
    go build -ldflags="$LDFLAGS" -o blockworld .
    PKG="blockworld-$VERSION-linux-$ARCH"
    STAGE="dist/$PKG"
    say "assembling $STAGE"
    rm -rf "$STAGE"
    mkdir -p "$STAGE"
    cp blockworld icon.png "$STAGE/"
    cat > "$STAGE/blockworld.desktop" <<DESK
[Desktop Entry]
Type=Application
Name=Blockworld
Comment=Mine by day, survive the night
Exec=blockworld
Icon=blockworld
Terminal=false
Categories=Game;
DESK
    cat > "$STAGE/install.sh" <<'INST'
#!/usr/bin/env bash
# Installs Blockworld for the current user with a menu entry and icon.
set -e
cd "$(dirname "$0")"
mkdir -p ~/.local/bin ~/.local/share/applications ~/.local/share/icons/hicolor/512x512/apps
cp blockworld ~/.local/bin/blockworld
cp icon.png ~/.local/share/icons/hicolor/512x512/apps/blockworld.png
sed "s|^Exec=.*|Exec=$HOME/.local/bin/blockworld|" blockworld.desktop > ~/.local/share/applications/blockworld.desktop
update-desktop-database ~/.local/share/applications 2>/dev/null || true
gtk-update-icon-cache ~/.local/share/icons/hicolor 2>/dev/null || true
echo "Installed. Find Blockworld in your application menu, or run ~/.local/bin/blockworld"
INST
    chmod +x "$STAGE/install.sh" "$STAGE/blockworld"
    (cd dist && tar -czf "$PKG.tar.gz" "$PKG")
    OUT="$STAGE/blockworld"
    ;;
  Windows)
    say "embedding icon and building"
    go-winres simply --icon icon.png --product-name Blockworld --file-description "Blockworld" \
      --product-version "$VERSION" --file-version "$VERSION"
    go build -ldflags="$LDFLAGS -H windowsgui" -o blockworld.exe .
    PKG="blockworld-$VERSION-windows-$ARCH"
    rm -rf "dist/$PKG"
    mkdir -p "dist/$PKG"
    cp blockworld.exe "dist/$PKG/"
    if command -v zip >/dev/null 2>&1; then
      (cd dist && rm -f "$PKG.zip" && zip -qr "$PKG.zip" "$PKG")
    elif command -v powershell >/dev/null 2>&1; then
      powershell -NoProfile -Command "Compress-Archive -Force -Path 'dist/$PKG' -DestinationPath 'dist/$PKG.zip'"
    fi
    OUT="dist/$PKG/blockworld.exe"
    ;;
esac

# ---------- checksums ----------
say "checksums"
(
  cd dist
  rm -f SHA256SUMS
  for f in *.zip *.tar.gz; do
    [ -f "$f" ] || continue
    if command -v shasum >/dev/null 2>&1; then shasum -a 256 "$f" >> SHA256SUMS
    elif command -v sha256sum >/dev/null 2>&1; then sha256sum "$f" >> SHA256SUMS
    fi
  done
  [ -f SHA256SUMS ] && cat SHA256SUMS || true
)

say "done"
ls -la dist | grep -v "^total\|^d.* \.\.\?$" | sed 's/^/    /'

if [ "$LAUNCH" = 1 ]; then
  say "launching"
  case "$OS" in
    Darwin) open "$OUT" ;;
    *) "$OUT" & ;;
  esac
fi

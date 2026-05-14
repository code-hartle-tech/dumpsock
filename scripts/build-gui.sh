#!/usr/bin/env bash
# Build the DumpSock GUI for the current host platform.
#
#   macOS  -> dist/DumpSock.app   (drag-droppable bundle, Finder treats as single file)
#   Linux  -> dist/dumpsock-gui   (single executable; AppImage packaging is Phase 4)
#   Windows: build via scripts/build.sh (cross-compile from any host)
#
# Usage:  bash scripts/build-gui.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VERSION="$(git describe --tags --dirty --always 2>/dev/null || echo dev)"
COMMIT="$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)"
LDFLAGS=(
    "-s" "-w"
    "-X" "github.com/code-hartle-tech/dumpsock/internal/branding.Version=${VERSION}"
    "-X" "github.com/code-hartle-tech/dumpsock/internal/branding.Commit=${COMMIT}"
)

mkdir -p dist

case "$(uname -s)" in
    Darwin*)
        APP="dist/DumpSock.app"
        rm -rf "$APP"
        mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

        # Ensure icon is built — build-icons.sh produces the .icns
        if [[ ! -f assets/icon/build/dumpsock.icns ]]; then
            echo "→ building icons (one-time)"
            bash scripts/build-icons.sh >/dev/null
        fi
        cp assets/icon/build/dumpsock.icns "$APP/Contents/Resources/dumpsock.icns"

        cat > "$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key><string>DumpSock</string>
    <key>CFBundleDisplayName</key><string>DumpSock</string>
    <key>CFBundleIdentifier</key><string>tech.hartle.dumpsock</string>
    <key>CFBundleVersion</key><string>${VERSION}</string>
    <key>CFBundleShortVersionString</key><string>${VERSION}</string>
    <key>CFBundleExecutable</key><string>DumpSock</string>
    <key>CFBundleIconFile</key><string>dumpsock.icns</string>
    <key>CFBundlePackageType</key><string>APPL</string>
    <key>CFBundleSignature</key><string>????</string>
    <key>LSMinimumSystemVersion</key><string>11.0</string>
    <key>NSHighResolutionCapable</key><true/>
    <key>NSPrincipalClass</key><string>NSApplication</string>
    <key>NSAppleEventsUsageDescription</key>
    <string>DumpSock uses Apple Events to show the output folder in Finder when you click Reveal.</string>
    <key>NSCameraUsageDescription</key>
    <string>DumpSock does not access the camera. (This entry is here only because some build chains require it.)</string>
    <key>LSApplicationCategoryType</key><string>public.app-category.utilities</string>
    <key>NSHumanReadableCopyright</key><string>© HARTLE.TECH</string>
</dict>
</plist>
PLIST

        echo "→ darwin/$(uname -m): go build → $APP/Contents/MacOS/DumpSock"
        # Wails v2.12 + macOS Tahoe (15+) needs UniformTypeIdentifiers
        # framework explicitly. See operator note in CLAUDE.md.
        CGO_ENABLED=1 \
        CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
            go build -trimpath -ldflags "${LDFLAGS[*]}" \
            -tags "desktop,production" \
            -o "$APP/Contents/MacOS/DumpSock" \
            ./cmd/dumpsock-gui

        # Mark as having been touched so Finder picks up the bundle.
        touch "$APP"

        echo
        echo "Built: $APP"
        du -sh "$APP" | awk '{print "  size:", $1}'
        ;;

    Linux*)
        OUT="dist/dumpsock-gui"
        echo "→ linux/$(uname -m): go build → $OUT"
        # webkit2_41 = webkit2gtk-4.1 (Ubuntu 22.04+/Fedora 36+).
        # For older systems, switch to plain `-tags desktop,production`.
        CGO_ENABLED=1 go build -trimpath -ldflags "${LDFLAGS[*]}" \
            -tags "desktop,production,webkit2_41" \
            -o "$OUT" \
            ./cmd/dumpsock-gui
        echo
        echo "Built: $OUT"
        du -sh "$OUT" | awk '{print "  size:", $1}'
        echo "  (AppImage / .desktop packaging: Phase 4)"
        ;;

    MINGW*|MSYS*|CYGWIN*)
        OUT="dist/DumpSock.exe"
        echo "→ windows: go build → $OUT"
        CGO_ENABLED=1 go build -trimpath \
            -tags "desktop,production" \
            -ldflags "${LDFLAGS[*]} -H windowsgui" \
            -o "$OUT" \
            ./cmd/dumpsock-gui
        echo
        echo "Built: $OUT"
        ;;

    *)
        echo "build-gui: unsupported host OS $(uname -s)" >&2
        exit 1
        ;;
esac

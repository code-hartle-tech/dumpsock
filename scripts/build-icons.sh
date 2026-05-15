#!/usr/bin/env bash
# Generate platform icon sets from the approved DumpSock mascot PNG.
#
# v2 (2026-05-15+): the source of truth is assets/brand/dumpsock_mascot_primary.png
# — provided by the operator as the approved design. We no longer use the
# hand-authored SVG (it's retained at assets/icon/dumpsock.svg for git
# history but never re-rendered).
#
# Outputs:
#   .icns for macOS    (assembled via iconutil from an .iconset/ dir)
#   .ico  for Windows  (multi-resolution via icotool)
#   .png  set          (Linux desktop entries, web favicons, embedded UI)
#
# Tooling: sips (built into macOS) + iconutil (macOS) + icotool
#          (brew install icoutils).
#
# Usage: bash scripts/build-icons.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC="$ROOT/assets/brand/dumpsock_mascot_primary.png"
OUT="$ROOT/assets/icon/build"
ICONSET="$OUT/dumpsock.iconset"

if [[ ! -f "$SRC" ]]; then
    echo "build-icons: missing source PNG at $SRC" >&2
    exit 1
fi
for bin in sips icotool; do
    if ! command -v "$bin" >/dev/null 2>&1; then
        echo "build-icons: missing dependency '$bin'" >&2
        case "$bin" in
            sips)    echo "  sips ships with macOS — are you on Linux?" >&2 ;;
            icotool) echo "  icotool: brew install icoutils" >&2 ;;
        esac
        exit 1
    fi
done

mkdir -p "$OUT" "$ICONSET"
rm -f "$OUT"/*.png "$OUT"/*.ico "$OUT"/*.icns "$ICONSET"/*.png

render() {
    local size="$1"
    local outfile="$2"
    sips -s format png -z "$size" "$size" "$SRC" --out "$outfile" >/dev/null
}

# 1) Generic PNG set
for size in 16 24 32 48 64 128 256 512 1024; do
    render "$size" "$OUT/dumpsock-${size}.png"
done

# 2) macOS .iconset → .icns
declare -a MACOS_PAIRS=(
    "16    icon_16x16.png"
    "32    icon_16x16@2x.png"
    "32    icon_32x32.png"
    "64    icon_32x32@2x.png"
    "128   icon_128x128.png"
    "256   icon_128x128@2x.png"
    "256   icon_256x256.png"
    "512   icon_256x256@2x.png"
    "512   icon_512x512.png"
    "1024  icon_512x512@2x.png"
)
for pair in "${MACOS_PAIRS[@]}"; do
    size="${pair%% *}"
    name="${pair##* }"
    render "$size" "$ICONSET/$name"
done

if command -v iconutil >/dev/null 2>&1; then
    iconutil -c icns "$ICONSET" -o "$OUT/dumpsock.icns"
    echo "  → $OUT/dumpsock.icns"
else
    echo "  (skipping .icns — iconutil not available; macOS-only tool)"
fi

# 3) Windows .ico (multi-resolution)
icotool -c \
    -o "$OUT/dumpsock.ico" \
    "$OUT/dumpsock-16.png" \
    "$OUT/dumpsock-32.png" \
    "$OUT/dumpsock-48.png" \
    "$OUT/dumpsock-64.png" \
    "$OUT/dumpsock-128.png" \
    "$OUT/dumpsock-256.png"
echo "  → $OUT/dumpsock.ico"

echo
echo "Done. Build outputs:"
ls -la "$OUT" | grep -v '^total' | awk '{printf "  %s  %s\n", $5, $NF}' | sort -n

# 4) Refresh the 256px preview used as a fallback in the readme/repo
cp "$OUT/dumpsock-256.png" "$ROOT/assets/icon/preview.png"
echo "  → assets/icon/preview.png (256px preview refreshed)"

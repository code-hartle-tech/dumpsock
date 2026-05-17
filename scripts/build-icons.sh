#!/usr/bin/env bash
# Generate platform icon sets from the DumpSock mascot SVG.
#
# v3 (2026-05-17): the icon is a tight PORTRAIT of the mascot's face —
# eyes + mouth + tongue + drool filling the square — with the cuff,
# stripes, sock outline, and body all cropped out. We render the SVG
# (vector) at high resolution and crop the face region in the SVG's
# native 460×650 reference frame, then downsample to each icon size.
# This keeps every size retina-grade crisp.
#
# v2 (2026-05-15, superseded): rendered from the approved mascot PNG
# (assets/brand/dumpsock_mascot_primary.png) with a top-anchored square
# crop that included the cuff + stripes + face. Replaced because the
# operator wants the face itself to fill the icon, with no sock context.
#
# Outputs:
#   .icns for macOS    (assembled via iconutil from an .iconset/ dir)
#   .ico  for Windows  (multi-resolution via icotool)
#   .png  set          (Linux desktop entries, web favicons, embedded UI)
#
# Tooling: rsvg-convert (brew install librsvg) + python3 + Pillow
#          + sips (macOS) + iconutil (macOS) + icotool (brew install icoutils).
#
# Usage: bash scripts/build-icons.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# Authored as a 1024×1024 square — straight tube-sock cuff with ribbing
# and two brand-red stripes. No cropping needed; rsvg-convert renders
# the square SVG directly at each target icon size.
SVG="$ROOT/assets/brand/dumpsock-icon.svg"
OUT="$ROOT/assets/icon/build"
ICONSET="$OUT/dumpsock.iconset"

for bin in rsvg-convert sips icotool; do
    if ! command -v "$bin" >/dev/null 2>&1; then
        echo "build-icons: missing dependency '$bin'" >&2
        case "$bin" in
            rsvg-convert) echo "  rsvg-convert: brew install librsvg" >&2 ;;
            sips)         echo "  sips ships with macOS — are you on Linux?" >&2 ;;
            icotool)      echo "  icotool: brew install icoutils" >&2 ;;
        esac
        exit 1
    fi
done
if [[ ! -f "$SVG" ]]; then
    echo "build-icons: missing source SVG at $SVG" >&2
    exit 1
fi

mkdir -p "$OUT" "$ICONSET"
rm -f "$OUT"/*.png "$OUT"/*.ico "$OUT"/*.icns "$ICONSET"/*.png

# Render each icon size DIRECTLY from the vector SVG via rsvg-convert.
# This is sharper than rasterising once and downsampling because every
# size gets a fresh vector pass with proper geometric anti-aliasing.
render() {
    local size="$1"
    local outfile="$2"
    rsvg-convert -w "$size" -h "$size" "$SVG" -o "$outfile"
}

# Keep a 1024 master PNG around for any downstream consumer that wants
# the raster form (build-gui.sh, README preview, embedded UI, …).
SQR="$OUT/dumpsock-master-square.png"
render 1024 "$SQR"
echo "  → master 1024px rendered from $SVG"

# 3) Generic PNG set
for size in 16 24 32 48 64 128 256 512 1024; do
    render "$size" "$OUT/dumpsock-${size}.png"
done

# 4) macOS .iconset → .icns
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

# 5) Windows .ico (multi-resolution)
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

# 6) Refresh the 256px preview used as a fallback in the readme/repo
cp "$OUT/dumpsock-256.png" "$ROOT/assets/icon/preview.png"
echo "  → assets/icon/preview.png (256px preview refreshed)"

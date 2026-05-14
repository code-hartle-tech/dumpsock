#!/usr/bin/env bash
# Cross-platform DumpSock build.
#
#   bash scripts/build.sh             # build all targets
#   bash scripts/build.sh darwin/arm64 linux/amd64   # build a subset
#
# Outputs: dist/dumpsock-<ver>-<os>-<arch>[.exe]
# Version is derived from `git describe` (falls back to short SHA + "dev").

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

ALL_TARGETS=(
    darwin/arm64
    darwin/amd64
    linux/amd64
    linux/arm64
    windows/amd64
)

TARGETS=("$@")
if [[ ${#TARGETS[@]} -eq 0 ]]; then
    TARGETS=("${ALL_TARGETS[@]}")
fi

mkdir -p dist

for triple in "${TARGETS[@]}"; do
    os="${triple%%/*}"
    arch="${triple##*/}"
    ext=""
    if [[ "$os" == "windows" ]]; then ext=".exe"; fi

    out="dist/dumpsock-${VERSION}-${os}-${arch}${ext}"
    echo "→ ${triple}  ${out}"

    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
        go build -trimpath -ldflags "${LDFLAGS[*]}" -o "$out" ./cmd/dumpsock
done

echo
echo "Done. Built:"
ls -lh dist/ | grep -v '^total' | awk '{printf "  %-10s  %s\n", $5, $NF}'

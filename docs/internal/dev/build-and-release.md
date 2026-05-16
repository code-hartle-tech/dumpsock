# Build & release

## Two binaries

DumpSock is two builds:

| Binary | Target | Script | CGo |
|---|---|---|---|
| `dumpsock` | CLI | `scripts/build.sh` | OFF |
| `DumpSock.app` (Mac) / `dumpsock-gui` (Linux / Windows) | GUI | `scripts/build-gui.sh` | ON |

The CLI is fully static and trivial to ship. The GUI carries CGo because Wails uses native webview bindings.

## CLI build

```bash
# scripts/build.sh
CGO_ENABLED=0 \
  go build \
  -trimpath \
  -ldflags="-s -w -X main.Version=$VERSION -X main.Commit=$(git rev-parse --short HEAD)" \
  -o dist/dumpsock \
  ./cmd/dumpsock
```

Matrix target tuples:
- `darwin/amd64`, `darwin/arm64`
- `linux/amd64`, `linux/arm64`
- `windows/amd64`

We build a fat universal binary for macOS with `lipo`:
```bash
lipo -create -output dist/dumpsock-darwin-universal \
     dist/dumpsock-darwin-amd64 dist/dumpsock-darwin-arm64
```

## GUI build (macOS)

```bash
# scripts/build-gui.sh
CGO_LDFLAGS="-framework UniformTypeIdentifiers -framework WebKit" \
  wails build \
  -clean \
  -tags "desktop,production" \
  -ldflags "-s -w -X main.Version=$VERSION" \
  -platform darwin/universal \
  -o DumpSock.app
```

Output: `dist/DumpSock.app`.

For ad-hoc local testing:
```bash
wails dev -tags "desktop"
```

## GUI build (Linux)

Requires:
- `webkit2gtk-4.0-dev`
- `libgtk-3-dev`

```bash
wails build -platform linux/amd64 -o dumpsock-gui-linux
```

## GUI build (Windows)

From a Mac via cross-build: not possible (Wails needs Windows toolchain). Build on a Windows runner in CI or on a real Windows box:

```bash
wails build -platform windows/amd64 -nsis -o DumpSock-Setup.exe
```

## Icons

`scripts/build-icons.sh` regenerates all platform icons from `assets/brand/dumpsock_mascot_primary.png`:

```bash
# macOS .icns
sips -z 16 16     master.png --out icon_16x16.png
sips -z 32 32     master.png --out icon_16x16@2x.png
sips -z 32 32     master.png --out icon_32x32.png
sips -z 64 64     master.png --out icon_32x32@2x.png
# ... up to 1024
iconutil -c icns DumpSock.iconset

# Windows .ico
icotool -c -o DumpSock.ico master-256.png master-128.png master-64.png master-32.png master-16.png

# Linux .png copies
# (Wails picks up build/appicon.png)
```

This script is idempotent — run it any time the master PNG changes.

## Embedding the frontend

`cmd/dumpsock-gui/main.go` uses `//go:embed`:

```go
//go:embed all:../../internal/frontend/dist
var frontendFS embed.FS

func mustSubFS() fs.FS {
    sub, err := fs.Sub(frontendFS, "../../internal/frontend/dist")
    if err != nil { panic(err) }
    return sub
}
```

Anything dropped into `internal/frontend/dist/` is in the binary on next build.

## Versioning

SemVer:
- `v0.X.Y` — pre-1.0, breaking changes allowed in `X`, fixes in `Y`.
- `v1.0.0` — first stable. Doesn't ship until: real Compare & Merge live, multi-device tested, Windows GUI smoke-tested.

`Version` is injected via `-ldflags`. The CLI's `version` subcommand prints both `Version` and the git SHA.

## Cutting a release

Full runbook: [Cut a release](../runbooks/release).

Quick path:
```bash
# 1. PR develop → main
# 2. tag main
git tag -a v0.2.0 -m "v0.2.0"
git push origin v0.2.0
# 3. build everything
VERSION=v0.2.0 scripts/build.sh
VERSION=v0.2.0 scripts/build-gui.sh
# 4. attach to GitHub release
gh release create v0.2.0 \
  dist/dumpsock-darwin-universal \
  dist/dumpsock-linux-amd64 \
  dist/dumpsock-windows-amd64.exe \
  dist/DumpSock.app.zip \
  --notes-file CHANGELOG-v0.2.0.md
```

## Reproducibility

`-trimpath -ldflags="-s -w"` plus a tag-locked `go.sum` mean two builds from the same commit should produce identical bytes. We don't verify this in CI yet (issue #6).

## Code-signing / notarization (Mac)

Not done yet. Pre-1.0 we ship un-notarized, and users do the Gatekeeper override. Pre-1.0 release notes call this out.

When we sign:
- Apple Developer ID Application certificate via HARTLE.TECH org enrollment.
- `codesign --options=runtime --timestamp --sign "Developer ID Application: ..." DumpSock.app`
- `xcrun notarytool submit DumpSock.app.zip --apple-id ... --keychain-profile ...`
- `xcrun stapler staple DumpSock.app`

Issue #5 tracks the Apple Developer enrollment paperwork.

## Windows signing

Same story — un-signed pre-1.0. EV cert via the org for v1.0.

## Linux

We don't sign. We do publish a SHA-256 for each tarball next to the GitHub release asset.

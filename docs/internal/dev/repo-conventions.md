# Repo conventions

## Golden rules (sacred, every commit)

1. **No secrets in any file.** Reference Cortex paths, never values. There are no secrets in DumpSock right now (no API, no telemetry), but if one ever lands, it goes in Cortex.
2. **Public identity = HARTLE.TECH + `contact@hartle.tech`.** Never the operator's name. Git author for this repo is `hartle-tech <noreply@hartle.tech>` (see `.git/config`).
3. **Per-project distinctness.** This repo inherits the org's GitHub + Cortex + Tailscale + CF setup; it CREATES its own GitHub repo, board, and Cortex subtree. Do not reuse another project's resources.

## Go version

`Go 1.24+`. Listed in `go.mod`. The CLI build uses `-trimpath -ldflags="-s -w"` for reproducibility.

## CGo policy

- **CLI build** (`scripts/build.sh`) → `CGO_ENABLED=0`. Static binary, no system deps.
- **GUI build** (`scripts/build-gui.sh`) → CGo on because Wails needs it. On macOS we link `UniformTypeIdentifiers` and `WebKit` via `CGO_LDFLAGS`.

## Branching

git-flow:
- `main` — last shipped release. Tagged. No direct commits.
- `develop` — integration branch. PRs merge here.
- `feature/<short-name>` — branch off `develop`, merge back via PR.
- `release/v0.X` — when stabilizing a release.
- `hotfix/<short>` — branched off `main`, merged into both `main` and `develop`.

## Commit message format

Narrative, atomic, `type(scope SXXEXX #NNN): subject`:

```
feat(gui S01E03 #4): live-poll for device disconnects every 2.5s

The dashboard kept showing a connected device after the cable was
unplugged. We now poll `ListDevices()` on a 2.5s interval whenever
no backup is active, and clear the device card if the UDID drops.

Polling pauses during an active backup (the backup itself holds the
AFC handle) and on visibilitychange=hidden to avoid burning cycles
when the window's in the background.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
```

- `type` — feat | fix | chore | refactor | docs | test | ci | brand
- `scope` — `cli`, `gui`, `afc`, `backup`, `exif`, `docs`, `brand`, `infra`
- `SXXEXX` — session/episode marker. `S01E01` = first session of season 1. We're currently on S01.
- `#NNN` — GitHub issue number this commit advances. If multiple, list the primary; mention others in the body.

## File naming

- Go files: `lower_snake.go` (not enforced by `gofmt` but standard).
- Markdown: `kebab-case.md`.
- SVG / PNG: `kebab-case-with-purpose.svg`. The canonical mascot is `dumpsock_mascot_primary.png` (underscore for historical reasons; do not rename without updating `scripts/build-icons.sh`).
- Tests: `<name>_test.go`, table-driven.

## Lint / format

- `gofmt -s -w ./...` before commit.
- `go vet ./...` clean.
- `golangci-lint run ./...` clean (config in repo root).
- Frontend: `prettier --write internal/frontend/dist/*.{html,css,js}` if we ever add it; currently hand-formatted.

## Imports

- `goimports` to sort.
- Group: stdlib, third-party, this-repo.
- Avoid wildcard imports.

## Testing

- Unit tests live next to the code (`backup_test.go` beside `backup.go`).
- Table-driven where it makes sense.
- No mocks for the AFC layer — instead, dependency-inject a small `Conn` interface and provide a memory-backed fake.
- The `internal/afc` package has hardware-dependent tests gated by `-tags=hardware` to keep CI green.

## Issue tracker

[GitHub Project 7 on `code-hartle-tech` org](https://github.com/orgs/code-hartle-tech/projects/7). Columns: Backlog / In Progress / In Review / Done. Issues are filed against the `dumpsock` repo.

## Docs

- Per-task lessons live here (internal wiki).
- Public-facing usage docs in `docs/external/`.
- `CLAUDE.md` at repo root holds house rules for AI collaborators.

## What ships in the binary

- The CLI binary embeds nothing extra.
- The GUI binary embeds `internal/frontend/dist/` via `//go:embed`.
- Brand assets (mascot SVG, PNG) live in `assets/brand/` and are copied into `internal/frontend/dist/` by `scripts/build-gui.sh`.

## What does NOT ship

- Tests, fixtures, README, docs, scripts.
- Any file in `.gitignore`'d paths.
- The Python MVP or any prototype scratch files.

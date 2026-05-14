# SESSION_RESUME — DumpSock

> One-paragraph close-of-session snapshot, refreshed every session. Goal: next session rebuilds full state in under a minute.

**Last updated:** 2026-05-14

---

## Where we are

**Phase 0 — scaffold:** ✅ complete (commit `5bc2647`).

**Phase 1 — Go CLI feature parity + icloudpd extras:** ✅ merged into `develop`.

**Phase 2 — Wails desktop GUI:** ✅ complete on `feature/gui-wails`. `dist/DumpSock.app` is a 9.3 MB drag-droppable bundle on macOS, charcoal HARTLE.TECH theme, shares the same `internal/backup` engine via an OnProgress callback that forwards to `runtime.EventsEmit("backup:progress")`. `scripts/build-gui.sh` produces it; needs `CGO_LDFLAGS="-framework UniformTypeIdentifiers"` and `-tags desktop,production` on macOS Tahoe.

| Command | Status | Notes |
|---|---|---|
| `dumpsock` (root) | ✅ | Brand-line help, version flag, minimal default surface |
| `dumpsock devices` | ✅ working against real iPhone | Uses go-ios; `--json` for machine output |
| `dumpsock version` | ✅ | Injects via `-ldflags` at build time |
| `dumpsock pull` | ✅ engine complete | Day-1 flags visible; advanced hidden behind --help |
| `dumpsock pull --dry-run` | ✅ tested on operator's iPhone | Walks DCIM, found 793 media files, listed correctly |

icloudpd-equivalent flags wired:

- `-o, --output` (default `~/DumpSock/<device-name>`)
- `--since / --until YYYY-MM-DD` (post-pull EXIF filter)
- `--delete-after` + `--confirm-delete` (hard gate; AFC rm wiring is v0.1)
- `--watch SECONDS` (daemon-style rescan loop, Ctrl-C clean)
- `--dry-run`
- `--udid` (multi-device disambiguation)

Advanced (hidden):
- `--parallel N`, `--until-found N`, `--no-mtime`, `--no-live-pair`, `--no-notify`, `--hash size|sha256`, `--remote-root PATH`, `--json`

Pipeline complete:

- `scripts/build.sh` produces all 5 cross-platform binaries (darwin arm64/amd64, linux amd64/arm64, windows amd64) — verified, ~8 MB each, statically linked
- `scripts/build-icons.sh` renders SVG → .icns / .ico / multi-size PNG set
- `.github/workflows/ci.yml` — gofmt + vet + build + race-test on push/PR to develop/main
- `.github/workflows/release.yml` — tag-push triggers matrix build + draft GitHub Release
- `internal/dedup/dedup_test.go` covers the 6 interesting branches of `PickPath`

## What's still open

| # | Item | Status |
|---|---|---|
| 1 | Operator confirms **repo name** (`dumpsock` / `dumpsock-cli` / other), **project board** (new #7 or reuse #6), **subdomain** | ⏳ awaiting |
| 2 | Push `develop` + feature branch to origin | blocked by #1 |
| 3 | `gh repo create` + project board entry + first issue | blocked by #1 |
| 4 | Final-quality icon artwork (current SVG is explicit v0 placeholder) | designer commission, future phase |
| 5 | Phase 2 — Wails GUI | future |
| 6 | Phase 3 — localhost-bridge browser UI | future |
| 7 | Phase 4 — codesigning, notarization, SBOM | future |
| 8 | Phase 5 — mobile-to-mobile native app | future |

## What's running

- Local commits on `feature/cli-skeleton` (off `develop`): `5bc2647`, `f8cf038`, `f9d2f30`, `b44cf70`, `aa4a328`, `0d84fdb`. All local, nothing pushed yet.
- Python `iphonepd.py` backup task `bb486wtjk` **failed** at 550/793 files because the Lexar SSD physically disconnected. 243 files erroring with `Permission denied: '/Volumes/Lexar'`. Drive needs to be replugged; the run is resumable (name+size dedup re-skips the 550 already on disk if the volume comes back intact).

## Next step on resume

1. **Operator answers the three open questions** above.
2. `gh repo create code-hartle-tech/<name> --private` (private until Phase 2).
3. `git -C ~/Projects/dumpsock remote add origin <url> && git push -u origin develop && git push -u origin feature/cli-skeleton`.
4. Open the first issue (`chore(scaffold): phase 0`), then `feat(pull): port iphonepd.py to Go` (already done — close on push), then a tracker for Phase 2 GUI.
5. Tag `v0.1.0` once feature/cli-skeleton merges into develop, and again once develop merges to main — that triggers the release pipeline and produces signed-by-GitHub binaries on a draft Release.

## Operator's 74 GB iPhone backup — practical recovery path

Once the Lexar reconnects:

```bash
ls /Volumes/Lexar/Backup/iCloud/Photos | head  # confirm prior 550 survived
bash /Volumes/Lexar/Backup/dumpsock            # Python wrapper, resumes via name+size dedup

# OR (now available):
~/Projects/dumpsock/dist/dumpsock-f9d2f30-darwin-arm64 pull -o /Volumes/Lexar/Backup/iCloud/Photos
```

Both tools target the same on-disk layout. Pick either; they're interchangeable.

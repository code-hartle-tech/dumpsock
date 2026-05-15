# DumpSock — Claude house rules

> Project codename: **DumpSock**. Local-first iPhone (and eventually Android) media-dump tool. CLI + future GUI + future browser-bridge UI. Single-file, portable, cross-platform, Apple-tier ease of use.
>
> Inherits all org-wide rules from https://void.neartrace.app/claude/handoff. This file only documents the **per-project** deviations and specifics.

---

## What this project is

A successor to `icloudpd` for the **local** path: phone-over-USB → storage media. No iCloud, no API keys, no 2FA flows. Just plug the phone in, run one command (or open one .app), get a tidy YYYY-MM-DD/ tree of photos and videos on disk.

**Why a new project, not a fork of icloudpd:** icloudpd is Python + iCloud-API-coupled. DumpSock is Go + AFC-coupled. Different language, different protocol, different scope — but identical organizational paradigm on disk so the output is interchangeable.

---

## Stack & build

| Layer | Tool | Why |
|---|---|---|
| Language | **Go 1.24+** | Single static binary across darwin/linux/windows arm64+amd64. Apple-tier "just-double-click" distribution requires no runtime, no installer. (Bumped from 1.23 to 1.24 when imagemeta landed; toolchain auto-downloads on first build.) |
| iOS device | **`github.com/danielpaulus/go-ios`** (pure-Go AFC + lockdownd) | No CGo / libimobiledevice. Cross-compiles cleanly. |
| EXIF | **`github.com/evanoberholster/imagemeta`** (HEIC/HEIF/JPG/PNG/DNG/CR2/CR3/NEF/ARW) + a hand-rolled ISO-BMFF `moov/mvhd` walker (MOV/MP4/M4V) + os.Stat mtime fallback | pure-Go, zero external dependency. Binary is fully self-contained. No `brew install exiftool` ever. |
| CLI | **`github.com/spf13/cobra`** | Sub-commands, Apple-style minimal surface up top, advanced flags grouped. |
| GUI (future) | **Wails v2** | Single binary; Go backend + HTML/CSS frontend; native window per OS. |
| Web UI (future) | localhost-bridge model | Browser cannot reach iPhone AFC directly. A running DumpSock process exposes a localhost API; the public page at `dumpsock.<domain>` talks to it. Phone-to-phone transfer = native mobile app, not web. |
| Test | `go test ./...` | Standard. Table-driven tests for dedup logic and date selection. |
| Lint | `golangci-lint` | Standard. |

**Build entrypoint:** `scripts/build.sh` produces all OS/arch binaries into `dist/`. CI mirrors this via `.github/workflows/release.yml` on tag push.

**No CGo for the CLI** — keeps `GOOS=windows GOARCH=amd64` cross-compile from macOS trivial. If a feature genuinely needs CGo (e.g., codesigning hooks), put it behind a build tag and ship as an "extended" binary.

---

## Identity & brand (inherited but worth restating because this is the first DumpSock session)

- **Public brand:** `HARTLE.TECH` only. Tagline acceptable: `DumpSock — a HARTLE.TECH tool` or `by HARTLE.TECH`.
- **Public contact:** `contact@hartle.tech`. **Never** the operator's name, **never** `rbfghrtl@gmail.com`.
- **Git author:** `hartle-tech <125872028+hartle-tech@users.noreply.github.com>` — NAME and email both anonymous.
- **No "Co-Authored-By: Claude" lines** in commits unless explicitly requested.
- **No `--no-verify` / `--no-gpg-sign` / `--amend`** of published commits without explicit ask.

---

## Visual / UX identity — v2 (light, mascot-driven, playful)

**Source of truth:** `assets/brand/` — approved PNG assets shipped by the operator on 2026-05-15. The v2 brief lives at `assets/brand/v2_brief.md`. Brand tokens at `assets/brand/brand_tokens.yaml`.

| | |
|---|---|
| **Mascot** | Straight hanging sock with thick black outline, ribbed cuff at the top, two red horizontal stripes near the cuff, white body with soft gray shading, cute manga/emoji-adjacent face (round eyes + smile), small red tongue + blue drool drop *inside* the mouth area only. No external water drops, sweat beads, or splash effects. Toe area implies slight dampness internally. |
| **Wordmark** | "Dump" in `#111111` ink, "Sock" in `#E62A28` red, Space Grotesk 800-weight, letter-spacing -0.04em. |
| **Tone** | Casual, witty, slightly tongue-in-cheek. **Useful first, joke second.** Cute and memorable, not crude. CTAs lean playful ("Dump it & back it up", "Wring it dry") but error messages stay precise. |
| **Voice rules** | Helpful exclamation marks OK in v2 (was banned in v1; lifted). Emoji OK in copy where they earn their keep. One-line error messages with the actionable fix appended. |
| **Colors** | Primary red `#E62A28`, soft red `#FF4444`, accent pink `#FF8A8A`, ink `#111111`, charcoal `#1F1F1F`, slate `#4B5563`, rib gray `#8E8E93`, sock shade `#F2F2F4`, white `#FFFFFF`, drool blue `#66D1FF`, bg `#FAFAF7`. |
| **Type** | Logo / headings: **Space Grotesk** (500/700/800). Body / UI: **Inter** (400/500/600/700). Code / logs: **JetBrains Mono**. All loaded from Google Fonts CDN in the embedded webview; the standalone CLI inherits the terminal. |
| **Theme** | **Light only** for v2. The earlier dark cyberpunk direction is retired. Do not regress into dark mode unless explicitly asked. |

**Approved assets (do not regenerate; these are authoritative):**

- `assets/brand/dumpsock_mascot_primary.png` — the mascot, 1024×1024-ish.
- `assets/brand/dumpsock_horizontal_logo.png` — mascot + wordmark side-by-side.
- `assets/brand/dumpsock_stacked_logo.png` — mascot above wordmark.
- `assets/brand/dumpsock_monochrome_logo.png` — single-color variant.
- `assets/brand/dumpsock_app_icons.png` — square-tile previews of platform-specific app icons.

**Mockups for downstream UI work** (Phase 2.1+):

`assets/mockups/` holds the operator's design-spec PNGs — clean mobile UI, playful desktop UI, brand guidelines, landing-page mock. Reference these when building the multi-tab dashboard, the storage gauge component, or the landing page.

**Icon spec (v2):**

- Primary: ship the approved mascot PNG directly to the embedded frontend (`internal/frontend/dist/dumpsock-mascot.png`) and as the macOS / Windows / Linux app-icon source.
- `scripts/build-icons.sh` converts the mascot PNG to `.icns` / `.ico` / multi-size PNG set via `sips` + `iconutil` + `icotool`. No SVG re-authoring — the PNG is the master.
- Old hand-authored SVG at `assets/icon/dumpsock.svg` is **deprecated**; kept for git history but never re-rendered.

---

## CLI surface (design intent)

Minimal day-to-day surface visible up front; advanced options grouped behind `--help-advanced` (or just hidden from default `--help` via Cobra's `Hidden: true`).

```
dumpsock                   # default: pull (if exactly one device) into ~/DumpSock/<device-name>/
dumpsock pull              # explicit
dumpsock devices           # list connected iPhones
dumpsock version
```

Day-1 flags on `pull`:

```
-o, --output DIR          # where to write (default: ~/DumpSock/<device-name>)
    --since YYYY-MM-DD    # only files captured on/after
    --until YYYY-MM-DD    # only files captured on/before
    --delete-after        # remove from device after verified write (HARD-GATED: requires --confirm-delete on the same line)
    --watch SECONDS       # daemon mode; rescan every N seconds while phone is plugged
    --dry-run
```

Advanced (hidden from default `--help`):

```
    --parallel N          # default 4
    --until-found N       # stop after N consecutive name+size matches (icloudpd-style)
    --no-mtime            # don't set file mtime to capture date
    --no-live-pair        # don't group HEIC+MOV live photo pairs
    --no-notify           # don't fire macOS notification on completion
    --hash {sha256,size}  # dedup mode; default size+name; sha256 = paranoid
    --remote-root PATH    # default DCIM; advanced: PhotoData, etc.
    --json                # machine-readable progress stream
```

**No flag may default to a destructive behavior.** `--delete-after` is always gated.

---

## File layout convention

Output mirrors icloudpd so the two tools are interchangeable on disk:

```
<output>/
  YYYY-MM-DD/
    IMG_XXXX.HEIC
    IMG_XXXX.MOV     # live-photo pair lives in the same date folder
    IMG_XXXX.DNG
    ...
  0000:00:00 00:00:00/   # files with no readable date
    ...
```

Date precedence (most authoritative wins): `DateTimeOriginal` > `CreateDate` > `MediaCreateDate` > `FileModifyDate`.

Dedup: name+size match anywhere under output → skip. Same name, different size, same date folder → append `-1`, `-2`, ...

---

## Repository conventions (per-project layer over org rules)

| | |
|---|---|
| **Default branch** | `develop` (work) and `main` (released, tagged). git-flow. |
| **Branch prefix** | `feature/* fix/* chore/* docs/* refactor/*` off `develop`. Release batches: `release/*` → `main` + tag. |
| **Commit format** | `type(scope SXXEXX #NNN): subject` once issues exist. Pre-issue scaffold commits: `chore(scaffold): subject`. |
| **License** | TBD — pending operator decision. No other `code-hartle-tech` repo carries a `LICENSE` file (all default to all-rights-reserved). Match that default until Phase 2 public-release time. |
| **Min Go** | 1.24 (declared in `go.mod`). |
| **Public-facing strings** | No operator name; no `rbfghrtl@`. Help text + version-string brand line: `DumpSock — by HARTLE.TECH · contact@hartle.tech`. |
| **CI** | GitHub Actions only. Matrix build on tag push. No external CI. |
| **Telemetry** | None. Zero phone-home. Document this in README. |

---

## Cortex / SSO / platform usage

- **Likely needed for v0:** none. DumpSock is offline-by-design.
- **Needed for release:** GitHub Actions release uploader uses the org `GITHUB_TOKEN` (already in Cortex at `secret/hartle.tech/github/pat`) — fetched in CI, never committed.
- **Future signing/notarization (Phase 4):** Apple Developer ID cert + notarization API key would live at `secret/dumpsock/apple/*`. Windows code-signing cert at `secret/dumpsock/windows/*`. Neither exists yet; do not pretend they do.
- **Subdomain:** TBD with operator. Candidates: `dumpsock.app` (own zone) or `dumpsock.hartle.tech` (subdomain on existing zone, less likely since this is a product brand).

---

## Non-goals

- **Cloud backup.** Use icloudpd for that. DumpSock is local-only.
- **iOS jailbreak features.** DumpSock uses the same AFC protocol Finder/iMazing use. No SSH-to-iPhone, no MobileBackup2 (that's `idevicebackup2`'s job).
- **Photo editing or library management.** Output is a folder of files. Apps like Photo Sync / PhotoSweeper take it from there.
- **Multi-user / cloud sync of backups.** This is a single-user, single-machine tool.

---

## Phase plan

| Phase | What lands | Status |
|---|---|---|
| **0** | This scaffold; rules nailed | in progress |
| **1** | Go CLI feature-parity with `iphonepd.py` + icloudpd extras (set-mtime, parallel, live-pair, until-found, --since/--until, notify) | next |
| **2** | Wails GUI; .app/.exe/.AppImage bundling; macOS-native progress + finder reveal | future |
| **3** | Localhost-bridge architecture; `dumpsock.app` static page talks to running CLI/GUI | future |
| **4** | Codesigning / notarization / signed installers / SBOM / SLSA provenance | future |
| **5** | Mobile-to-mobile: native Android app that ingests iPhone-over-OTG | future |

Each phase ships its own release / batch; don't conflate them.

---

## Anti-patterns specific to this project

- **Do not** silently overwrite files at destination. Always dedup-and-suffix or skip.
- **Do not** mutate files on the device. (Phase 1 is read-only. Phase 1.5 may add `--delete-after`, hard-gated.)
- **Do not** prompt for unlock / pairing on macOS via shell — pairing must happen in Finder / Settings, the binary just consumes existing pair records.
- **Do not** require `exiftool` (or any external binary) at runtime. EXIF reading is pure-Go via imagemeta + a built-in moov-atom walker. The bundle is a true single file.
- **Do not** add emoji to CLI default output. Voice rule above.

---

## When stuck — DumpSock-specific reference

| Question | Read |
|---|---|
| What does the on-disk layout look like? | This file → "File layout convention" |
| What flags are user-facing vs hidden? | This file → "CLI surface" |
| Why Go and not Python? | This file → "Stack & build" |
| Where do we deviate from icloudpd? | This file → "Non-goals" + "CLI surface" |
| Where do per-project Cortex secrets go? | Handoff doc; convention is `secret/dumpsock/*` |
| What's the release flow? | Org handoff: `feature → develop → release → main → tag` |

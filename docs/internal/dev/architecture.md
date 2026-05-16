# Architecture

## 30-second pitch

```
┌─────────────────────────────────────────────────────┐
│ Wails GUI (Go + HTML/CSS/JS webview)                │
│  cmd/dumpsock-gui/main.go                           │
│  internal/gui/app.go        ← bindings exposed to JS │
│  internal/frontend/dist/    ← HTML, CSS, mascot SVG │
└──────────────────────┬──────────────────────────────┘
                       │ JSON over Wails IPC
┌──────────────────────▼──────────────────────────────┐
│ internal/backup        ← orchestrator (engine)      │
│   ├─ Options, Result, ProgressEvent                 │
│   ├─ Run()             ← phases: index/walk/plan/   │
│   │                       pull/delete/done          │
│   └─ runDeletions()    ← worker pool                │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│ internal/afc           ← go-ios wrapper             │
│   ├─ Open/Close                                     │
│   ├─ Walk, PullTo                                   │
│   ├─ Remove (delete-after)                          │
│   └─ DeviceStorage                                  │
├─────────────────────────────────────────────────────┤
│ internal/exif          ← pure-Go EXIF + moov walker │
└─────────────────────────────────────────────────────┘
                       │
                       ▼
                  usbmuxd (system)
                       │
                       ▼
                    iPhone (AFC service @ /var/mobile/Media/)
```

## Why we picked these pieces

### Go single binary

- No runtime install on user's machine.
- Cross-compiles for macOS + Linux + Windows.
- Easy to ship via Homebrew / Scoop / direct download.
- Native concurrency primitives match the I/O pattern (one walker, N pullers, one progress sink).

### Wails for the GUI

- Webview-based — we can iterate on UI in HTML/CSS without leaving Go.
- No Electron weight; the binary is ~40 MB instead of ~200.
- Cross-platform out of the box.
- Easy to put a polished UI on top without learning native macOS / WinUI / GTK.

### go-ios over libimobiledevice

- Pure Go — no CGo for the CLI build (CGo is unavoidable for the GUI because Wails has C bindings, but the CLI build is fully static).
- Active maintenance, iOS 17+ support, AFC + lockdownd + service discovery all in one library.
- The original Python MVP used `pymobiledevice3`, but that meant shipping Python with the app — go-ios is a much cleaner story.

### Webview frontend, not React/Vue

- The UI is small enough that vanilla JS keeps the dist folder tiny and the load instant.
- No build step in the frontend → easier to embed via `//go:embed`.
- Brand-token-driven CSS is auditable as a single file.
- If the UI grows we can swap in any framework — the Wails binding layer is framework-agnostic.

## Module layout

```
dumpsock/
├── cmd/
│   ├── dumpsock/              ← CLI entry, Cobra root
│   └── dumpsock-gui/          ← Wails entry, embeds frontend
├── internal/
│   ├── afc/                   ← go-ios wrapper
│   ├── backup/                ← orchestrator
│   ├── exif/                  ← imagemeta + moov walker
│   ├── frontend/dist/         ← embedded webview assets
│   └── gui/                   ← Wails App, bindings, config
├── assets/brand/              ← canonical PNG + SVG mascot, brand-bible.md
├── scripts/                   ← build.sh, build-gui.sh, build-icons.sh
├── docs/
│   ├── external/              ← public VitePress
│   └── internal/              ← this wiki
└── CLAUDE.md                  ← per-project house rules
```

## Data flow for a single pull

1. **User opens GUI**, Wails `OnStartup` runs → loads `~/Library/Application Support/DumpSock/config.json`.
2. **JS bootstrap** calls `App.ListDevices()`. Cards rendered.
3. **User clicks "Start backup"** → JS calls `App.StartBackup(opts)`.
4. **`gui.App.StartBackup`** spawns a goroutine wrapping `backup.Run(ctx, opts, emit)`.
5. **`backup.Run`** emits `ProgressEvent`s through `emit`, which the App forwards to JS via Wails `runtime.EventsEmit("backup:progress", event)`.
6. **`afc.Walk`** enumerates `/var/mobile/Media/DCIM/*APPLE/*` over the AFC service.
7. **Local index** is built by walking the destination tree once at start (`map[nameSize]path`).
8. **Plan** = remote ∖ local index (by name+size, optionally hash).
9. **N puller goroutines** consume the plan, call `afc.PullTo` for each file, emit `"pulling"` events on start of each pull.
10. **After each pull**: `exif.Read` extracts capture date, file moves to `<output>/<YYYY-MM-DD>/<name>`.
11. **If `--delete-after`**: pulled file's name + size verified against remote, then queued for `runDeletions`.
12. **Phase = "done"** emitted at end, JS swaps to Summary modal.

## Phases (the strings emitted by `backup.Run`)

- `indexing` — walking destination, building dedup map
- `walking` — enumerating remote DCIM
- `planning` — computing diff
- `pulling` — actively transferring (per-file progress)
- `deleting` — running device-side removal (only with `--delete-after`)
- `done` — summary ready
- `error` — fatal error (`event.Err`)
- `canceled` — context canceled, partial result available

The GUI uses these to swap the dashboard's main panel. The CLI just prints them as plain text.

## What's NOT in this repo

- The original Python icloudpd-2FA-workaround script (lost / never committed).
- The pymobiledevice3 MVP (superseded; not migrated).
- iPhone Mirroring + CGEventPost prototype (see [discoveries/recently-deleted-cannot-be-bypassed](../discoveries/recently-deleted-cannot-be-bypassed) — punted to v0.2).

# GUI structure

The GUI is a Wails v2 app: Go binary hosting a webview with our HTML/CSS/JS bundle embedded.

## Files

```
internal/
├── gui/
│   ├── app.go              ← bindings (methods exposed to JS)
│   ├── config.go           ← atomic persistence to ~/Library/Application Support/DumpSock/config.json
│   └── reveal_<os>.go      ← platform-native "show in Finder/Explorer/file manager"
├── frontend/
│   └── dist/               ← embedded via //go:embed
│       ├── index.html      ← shell with chrome bar + sidebar + content
│       ├── style.css       ← v3 tokens, animations
│       ├── main.js         ← all client logic
│       └── dumpsock-mascot.svg
└── cmd/dumpsock-gui/main.go ← Wails entry, embed, options
```

## Layout in `index.html`

```
┌──────────────────────────────────────────────────────────┐ ← .chrome (36px high, draggable, traffic-lights live in first ~80px)
│ ⬤⬤⬤  [88px gap] DumpSock                                │
├──────────────────────────────────────────────────────────┤
│ .sidebar (240px wide)   │ .content (flex-grow)           │
│  ┌────────────────────┐ │ ┌───────────────────────────┐  │
│  │ 🧦 DumpSock        │ │ │ <header>                  │  │
│  ├────────────────────┤ │ │  Title + actions          │  │
│  │ Dashboard          │ │ ├───────────────────────────┤  │
│  │ Compare & Merge    │ │ │ <main> .tab-panels        │  │
│  │ Backups            │ │ │  [data-tab=dashboard]     │  │
│  │ Logs               │ │ │  [data-tab=compare]       │  │
│  │ Settings           │ │ │  [data-tab=progress]      │  │
│  │                    │ │ │  [data-tab=backups]       │  │
│  │      ▼ bottom      │ │ │  [data-tab=logs]          │  │
│  │ Help · About       │ │ │  [data-tab=settings]      │  │
│  └────────────────────┘ │ └───────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

The chrome bar uses `--webkit-app-region: drag` so the user can grab anywhere in the empty area and drag the window. The traffic lights need clearance; we pad the leading content by 88 px.

## Tab switching

`<body>` has `data-tab="<id>"`. CSS hides every panel except `[data-tab=<id>]` via `body[data-tab=<id>] .panel:not([data-tab-target=<id>])  { display: none; }`. JS handles sidebar clicks:

```js
sidebarItem.addEventListener("click", () => {
  body.dataset.tab = sidebarItem.dataset.target;
  // notify panel-specific hooks
  if (body.dataset.tab === "logs") refreshLogs();
});
```

`hidden` is also used for transient elements (modals). Pitfall: CSS overriding `[hidden]` defeats it. We add `[hidden] { display: none !important; }` defensively, see [feedback_html_hidden_attribute_vs_css_display](https://github.com/code-hartle-tech/dumpsock) for the lesson.

## State (client-side)

```js
const state = {
  config: null,        // mirrors ~/Library/Application Support/DumpSock/config.json
  devices: [],         // last poll from ListDevices()
  selectedDevice: null,
  output: "",          // chosen destination
  backupActive: false, // gates polling pauses
  progress: null,      // last ProgressEvent
  logs: [],            // ring of recent events for the Logs tab
};
```

No framework — vanilla JS, hand-rolled DOM updates. Total `main.js` is ~600 LOC. If we cross ~1500 LOC, swap to Lit or Solid.

## Wails bindings (server → client)

`internal/gui/app.go` defines an `App` struct. Wails reflects every public method into `window.go.gui.App.MethodName(...)`. Current surface:

```go
func (a *App) Info() Info                                       // version, commit, OS info
func (a *App) GetConfig() Config                                // last output dir, settings
func (a *App) SaveLastOutput(p string) error
func (a *App) ListDevices() []Device                            // sorted by name
func (a *App) DefaultOutputFor(udid string) string              // ~/DumpSock/<name>
func (a *App) PickDirectory(starting string) (string, error)    // native folder picker
func (a *App) StartBackup(opts BackupOptions) error             // fires backup.Run() in goroutine
func (a *App) CancelBackup() error                              // cancels the run's ctx
func (a *App) RevealInFinder(path string) error
func (a *App) DeviceStorage(udid string) (Storage, error)       // total/free bytes from AFC
func (a *App) RunCompare(udid, output string) (CompareResult, error)  // TODO: #2 — placeholder today
```

## Wails events (server → client)

```go
runtime.EventsEmit(ctx, "backup:progress", event)
runtime.EventsEmit(ctx, "backup:done", result)
runtime.EventsEmit(ctx, "backup:error", err.Error())
runtime.EventsEmit(ctx, "device:changed", devices)  // TODO: fire from a usbmuxd subscription instead of polling
```

JS listens with `window.runtime.EventsOn("backup:progress", fn)`.

## Device freshness — the polling story

`device:changed` is not yet wired to a real usbmuxd notification stream. Until then, JS polls `ListDevices()` every 2.5 seconds. Pauses kick in:
- During an active backup (the backup itself holds the AFC handle; double-querying is wasteful).
- On `document.visibilitychange === "hidden"` (window unfocused / minimized).

This was [issue raised live by the operator](../sessions/2026-05-kickoff#device-disconnect): the dashboard kept showing a connected device after unplug. Polling fixed it; the deeper fix (subscribe to usbmuxd's `Listen` notifications) is filed as issue #7.

## Config persistence

`internal/gui/config.go`:
- Path: `~/Library/Application Support/DumpSock/config.json` on macOS; `$XDG_CONFIG_HOME/dumpsock/config.json` on Linux; `%APPDATA%\DumpSock\config.json` on Windows.
- Atomic write: write to `config.json.tmp`, `fsync`, then `os.Rename`.
- `GetConfig()` lazy-loads from disk every call. This avoids a race we hit early on: Wails `OnStartup` would load config asynchronously, JS bootstrap would call `GetConfig()` before the load completed, get an empty config, save empty config back over the real one.

## Building

`scripts/build-gui.sh` — see [Build & release](./build-and-release).

## Dev loop

```bash
wails dev -tags "desktop"
```

This watches Go files for changes and rebuilds. Frontend changes (HTML/CSS/JS) are hot-swapped via the dev server without a Go rebuild.

## Theming

There is no dark mode yet. All v3 tokens are designed for light. Dark mode is filed as issue #8 — would require a `[data-theme=dark]` token override block plus a wrinkle to the mascot (which has dark eyes that disappear on dark BG).

## A11y

- All buttons have keyboard focus rings (`:focus-visible` outline).
- Tab key cycles through sidebar → main content → modals.
- Color contrast ratios checked against AA at minimum. Primary CTA passes AA at body sizes.
- No screen reader audit yet — issue #9.

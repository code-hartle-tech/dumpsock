# Privacy

## What DumpSock does

- Reads photos / videos from your iPhone over USB.
- Writes them to a folder you chose.

## What DumpSock doesn't do

- Doesn't connect to any DumpSock servers (there are none).
- Doesn't connect to any third-party analytics, crash reporter, or telemetry service.
- Doesn't include any auto-update check.
- Doesn't read anything off your iPhone outside `/var/mobile/Media/DCIM/`.
- Doesn't ask for an account, email, or any sign-up step.
- Doesn't log your photo filenames, EXIF metadata, or device identifiers anywhere outside the destination folder.

## What DumpSock requires

- A USB cable to your iPhone.
- The iPhone to be trusted (you tap "Trust This Computer" on the phone once, ever, per Mac).
- macOS (today). Linux and Windows CLI builds too. The GUI requires `usbmuxd` (built into macOS, available on Linux, ships with Apple Devices on Windows).

## Network behavior in detail

The DumpSock binary itself makes **zero network calls** during normal operation. The only network traffic associated with the app is:

1. **Google Fonts CDN** — the GUI's webview loads three font families (SF Pro / Inter / JetBrains Mono) from `fonts.googleapis.com` and `fonts.gstatic.com` on first open. Cached after that. If you want to fully air-gap, build DumpSock with the fonts vendored locally (one-line patch in `frontend/dist/index.html` — see internal wiki).
2. **Mascot SVG** — bundled in the binary; no network call.
3. **GitHub Releases** — only when you manually click "Check for updates" (not yet implemented; will be opt-in when it ships).

You can verify this with `Little Snitch` or `lsof -i -P -n | grep DumpSock` — outside the optional fonts request, the process opens no sockets.

## Source available

[github.com/code-hartle-tech/dumpsock](https://github.com/code-hartle-tech/dumpsock). MIT license once we're out of pre-release (current status: private repo during beta).

## Reproducible builds

DumpSock binaries are built from tagged commits with `-trimpath -ldflags="-s -w"` so the same source produces the same binary. The version-string printed by `dumpsock version` includes the short SHA so you can verify which commit a given binary was built from.

## Reach us

`contact@hartle.tech`. We answer.

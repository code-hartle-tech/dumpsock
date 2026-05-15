# DumpSock

> Cloudless freedom for your iPhone.

DumpSock is a playful, local-first tool that helps people free space on iPhone and iPad by backing up media and files the good old way: **locally, privately, no cloud**.

```
dumpsock pull
```

That's the whole common case.

---

## What it does

- Plug your iPhone in over USB.
- DumpSock walks `DCIM/`, pulls every photo and video to your Mac (or to any folder you point it at — including an external SSD).
- Files land in date-sorted `YYYY-MM-DD/` folders so you can find them later.
- Optional `--delete-after` removes each file from the iPhone once it's safely on disk. Storage freed immediately.

No iCloud. No account. No telemetry. No phone-home. The bytes go from your phone to your disk and nowhere else.

## Status

**Phase 2 — desktop GUI shipped, v2 brand applied.** macOS `.app` bundle works end-to-end against real iPhones on iOS 26. Linux + Windows CLI binaries build; native GUIs for those platforms are in flight.

## Roadmap

| Phase | Lands |
|---|---|
| 0 | Project scaffold + brand bible |
| 1 | Go CLI — pull, devices, dry-run, set-mtime, parallel, live-pair, since/until, watch-mode, delete-after |
| 2 | macOS GUI (Wails) with the v2 mascot, light theme, playful copy |
| **2.1** | **Multi-tab dashboard (Dashboard / Compare / Progress / Settings), storage gauge, status pills** |
| 2.5 | iPhone Mirroring auto-empty of Recently Deleted (one Touch ID on the Mac, then bytes really go) |
| 3 | Landing page at `dumpsock.app` (or `dumpsock.hartle.tech`); browser-control of a desktop bridge |
| 4 | Signed installers, notarization, SBOM, SLSA provenance |
| 5 | Mobile-to-mobile (iPhone → Android over OTG, native) |

## Privacy

DumpSock is fully offline. No telemetry. No phone-home. No account. Your data stays yours.

## Contact

A HARTLE.TECH tool. Questions: `contact@hartle.tech`.

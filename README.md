# DumpSock

> Plug your phone in. Wring it dry.

DumpSock is a local, offline iPhone media-dump tool. One executable. No iCloud, no API keys, no 2FA dance. Connect, run, get a tidy date-sorted folder of photos and videos on disk.

```
dumpsock pull
```

That's the whole common case.

---

## Status

**Phase 0 — scaffold.** Project bootstrapped. CLI not yet building. Subscribe to releases for the first usable cut.

## Why

`icloudpd` is excellent for cloud-side backups. It does not exist for the local-USB path. DumpSock fills that slot: same date-folder output layout, but the bytes come over the USB cable, not the iCloud API.

## Roadmap

| Phase | Lands |
|---|---|
| 0 | Project scaffold |
| 1 | Go CLI: pull / devices / dry-run / set-mtime / parallel / live-pair / since-until / watch-mode |
| 2 | Native GUI (macOS / Windows / Linux), single-file `.app` / `.exe` / `.AppImage` |
| 3 | Browser UI via localhost bridge — `dumpsock` on the desktop, control surface in a tab |
| 4 | Signed installers, notarization, SBOM |
| 5 | Mobile-to-mobile (iPhone → Android over OTG, native) |

## Contact

A HARTLE.TECH tool. Questions: `contact@hartle.tech`.

## Privacy

DumpSock is fully offline. No telemetry. No phone-home. No account. Your photos move from your phone to your disk, and nowhere else.

# AFC + PTP behavior

The iPhone exposes media to host computers over two distinct USB interfaces:

| Service | What | Where files live | Free-space behavior |
|---|---|---|---|
| **AFC** (Apple File Conduit) | File-level access | `/var/mobile/Media/` (sandbox) | Deleting frees DCIM bytes, but Photos.sqlite still references them |
| **PTP** (Picture Transfer Protocol) | Image-protocol access (via libgphoto2-style verbs) | Photo library abstraction | Deleting goes through PhotoKit → Recently Deleted, no free space yet |

## What DumpSock uses

**AFC only**, via `github.com/danielpaulus/go-ios`.

```go
// internal/afc/afc.go
conn, _ := afc.New(deviceUDID)
fr, _ := conn.OpenRead("DCIM/100APPLE/IMG_0001.HEIC")
io.Copy(localFile, fr)
conn.Close(fr)
```

We chose AFC because:
1. **Bulk reads are fast.** AFC has a single-roundtrip read for known offsets/sizes.
2. **PTP has weird ordering issues** for video files (and zero benefit for our use case).
3. **PTP needs `usbmuxd` to expose a second device endpoint** — unreliable on some iPhones we tested.

## AFC paths we touch

```
/var/mobile/Media/
├── DCIM/
│   ├── 100APPLE/   ← read; this is where camera roll lives
│   ├── 101APPLE/
│   └── …
├── PhotoData/     ← NOT touched; this is Photos.app's sqlite, plist, thumb cache
├── Recordings/    ← Voice Memos; not currently pulled
└── ...
```

## What we read

Everything matching the media extension allow-list under `DCIM/*APPLE/`:
```
.heic .heif .jpg .jpeg .png .mov .mp4 .m4v .dng .raw .gif .webp
```

We do NOT read:
- `.AAE` sidecars by default (metadata, not media). They ARE deleted alongside media files when `--delete-after` is on.
- Anything outside `DCIM/`.
- `PhotoData/` (the sqlite database is sandboxed anyway; AFC can't reach it).

## What we delete (with `--delete-after`)

The same files we just successfully pulled, **plus matching `.AAE` sidecars**, via `AFC_OP_REMOVE_PATH`.

```go
func (c *Conn) Remove(remotePath string) error {
    return c.session.RemovePath(remotePath)
}
```

## What deletion does NOT do

This is the load-bearing finding from a multi-agent investigation [see "Recently Deleted cannot be bypassed"](../discoveries/recently-deleted-cannot-be-bypassed):

- **AFC delete frees DCIM bytes immediately.** The file is gone from `/var/mobile/Media/DCIM/*APPLE/`. Total device free space DOES go up.
- **Photos.sqlite still has the ZASSET row.** Photos.app still shows a "ghost" thumbnail with an exclamation mark, because the database row says "I have a photo!" but the file is missing.
- **The ghost is harmless** — tap it once and Photos.app silently removes the orphaned database row. It's not removed in bulk because Photos.app does its sqlite cleanup lazy.

## Why we don't go through PhotoKit

PhotoKit (the proper Apple API) DOES coordinate with the sqlite database and DOES not leave ghosts. But PhotoKit is only callable from on-device code (an iOS app) or from a Mac mirroring session — there's no USB API.

We empirically tested whether **PTP** deletion goes through PhotoKit (it could in principle). Result: on iOS 26.4.2, PTP DeleteObject **routes through Recently Deleted**, doesn't free bytes for 30 days, AND still leaves a different kind of ghost. Worse than AFC. See [PTP delete still trashes](../discoveries/ptp-delete-still-trashes).

## The Mirroring + CGEventPost prototype

The one viable bypass we found: launch iPhone Mirroring on macOS, drive the mirrored UI via `CGEventPost`, let macOS Touch ID substitute for the iPhone's Face ID confirmation prompt that pops up on bulk-delete-from-Recently-Deleted. Filed as issue #1. Prototyped to "works on my machine" stage; not productionized because:
- Fragile (Apple can break Mirroring auto-confirm any update).
- Apple-API-adjacent (gray area for shipping).
- Needs accessibility entitlement, which means Apple Developer ID notarization first.

Punted to v0.2.

## go-ios specifics

- The `afc` service is `com.apple.afc`. There's also `com.apple.afc2` (full filesystem) which requires jailbreak; we never use it.
- Connection pool: we open one AFC session per device per backup run. Multiple goroutines share the session; go-ios serializes the wire protocol internally.
- Storage info comes from `afc.GetDeviceInfo()` → keys `TotalBytes` / `FreeBytes`.

## Known go-ios bugs / quirks we work around

- `Walk` returns directory entries in non-deterministic order. We sort by path before queuing for puller goroutines so progress is reproducible.
- On iOS 17+, opening too many AFC sessions concurrently causes the service to time out the next open for ~30s. We cap to one session per backup.
- `Remove` returns success even for paths that don't exist. We verify with a follow-up `Stat` for files we expect to be gone.

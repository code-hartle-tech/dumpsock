# AFC delete leaves ghosts

## TL;DR

`AFC Remove` on a photo deletes the file from `/var/mobile/Media/DCIM/` but does NOT update `PhotoData/Photos.sqlite`. The result: bytes are freed, Photos.app shows a ghost thumbnail with an exclamation mark in the bottom-right corner. Tapping the ghost makes Photos.app remove the orphan database row silently. There is no USB-side fix; this is just a behavioral quirk users need to know about.

## What the user sees

After running `dumpsock pull --delete-after`:

1. Photos.app's All Photos view: a thumbnail with `!` in the corner. Tap → "This photo is unavailable" message → it disappears.
2. iPhone Settings → Storage → free bytes goes UP by the size of the deleted files within ~1 minute.

So the user gets the storage benefit (the whole point of `--delete-after`) but pays a transient ugliness in the Photos UI.

## Why it happens

Apple's Photos library is two coupled layers:

| Layer | Where | Purpose |
|---|---|---|
| Files | `/var/mobile/Media/DCIM/*APPLE/IMG_*.{HEIC,MOV,…}` | The actual photo + video bytes |
| Database | `/var/mobile/Media/PhotoData/Photos.sqlite` | Index of "this photo exists, here's its metadata, here's its album, …" |

AFC can reach the **files** (sandboxed access to `Media/DCIM/`). AFC CANNOT reach `PhotoData/` (sandboxed out).

When we AFC-delete a file:
- The byte is gone from disk → free space rises.
- The `Photos.sqlite` row (`ZASSET` table) still references the now-missing file.

When Photos.app next does its scan, it notices the missing file, marks the asset as "available offline = false / cloudPlaceholderKind = …". The exclamation-mark thumbnail is its way of saying "I think I have this photo but I can't find the file".

## The ghost cleanup

The user can clear ghosts three ways:

1. **Tap once.** Photos.app silently removes the orphan row on-tap.
2. **Wait.** Photos.app does a lazy garbage-collect on app launch + iCloud sync. Usually all ghosts are gone within 24h.
3. **Force a sync cycle.** Toggle iCloud Photos off → on. Photos.app rebuilds, drops orphans.

We document method 1 in the public docs.

## Why we don't fix this in DumpSock

Because we **can't**. The database is sandboxed. AFC2 (jailbreak AFC) could reach it; we don't support jailbroken devices.

The only Apple-blessed paths to update `Photos.sqlite`:
- An iOS app on-device using PhotoKit (irrelevant — we're a Mac/Linux/Windows app).
- iPhone Mirroring + scripted on-device taps (see [Recently Deleted can't be bypassed](./recently-deleted-cannot-be-bypassed) — deferred to v0.2).

## What does NOT happen

A common worry: "If the database thinks the photo still exists, will iCloud re-upload it from somewhere or restore the file?" No. iCloud Photos syncs files-to-DB consistency from the device side; if the database row is broken and the file is missing, iCloud marks the asset as deleted on next sync. The bytes don't come back.

## Code

- `internal/afc/afc.go:Remove()` — the AFC-side delete that triggers this.
- `internal/backup/backup.go:runDeletions()` — uses `Remove`.

There's no code that touches `Photos.sqlite`. We don't have access to it.

## Doc pointer for users

[Free up iPhone storage / After delete-after, tidy Photos.app](../../external/guide/free-storage#after-delete-after-tidy-photos-app)

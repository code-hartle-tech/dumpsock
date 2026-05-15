# FAQ

## Does this work without an iCloud account?

Yes. DumpSock has no iCloud component. The iPhone doesn't even need to be signed into iCloud Photos.

## Does this back up everything?

No. DumpSock backs up the **camera roll** (`/var/mobile/Media/DCIM/`) — photos, videos, Live Photo MOV components, raw DNG, screenshots. It does **not** back up:

- Messages
- App data (Notes, Voice Memos, Health, etc.)
- Settings
- Wi-Fi passwords / Keychain
- Music or books you didn't buy

For a full system backup, use Apple's encrypted backup via Finder or `idevicebackup2`.

## What about iCloud-only photos?

If you've enabled **Optimize iPhone Storage** in iCloud Photos, your full-res originals may not be on the device — only thumbnails. DumpSock pulls what's physically present at `/var/mobile/Media/DCIM/`. To get the full-res originals before backing up: Settings → Photos → toggle **Download and Keep Originals**, wait for sync, then run DumpSock.

## Does it work with iPad?

Yes. Same protocol, same flow.

## Why not just use AirDrop?

AirDrop is fine for a few files. For thousands, it's painfully slow, the receiver Mac has to be awake and discoverable, and there's no date-sorted output — everything lands in `~/Downloads/` with cryptic names. DumpSock does the bulk-archive case AirDrop doesn't.

## Does it have a Windows / Linux GUI?

CLI works on Linux and Windows today. Native GUIs for those platforms are on the roadmap. For now: the CLI handles every flow the GUI does.

## Will my photos appear in DumpSock's UI?

The Dashboard's Storage Overview shows aggregate iPhone storage and a category breakdown. It does **not** display thumbnails of your photos — DumpSock is a transport tool, not a photo viewer. Use Photos.app, [PhotoSweeper](https://overmacs.com/), [Lightroom](https://www.adobe.com/products/photoshop-lightroom.html), or your favorite tool to browse the resulting folder.

## What if a backup fails partway?

DumpSock is idempotent. Re-run the same command and it picks up where it left off — files already on disk are skipped (matched by filename + byte size).

## What if I want a copy of the same photo set in two places?

Run two DumpSock pulls, each with a different `--output` dir. They're independent and won't trample each other.

## Is `--delete-after` reversible?

**No.** Once a file is removed from `/var/mobile/Media/DCIM/`, it's gone from the iPhone's filesystem. The Photos.app database may still show a ghost entry for a while, but the underlying bytes are off the device. Always verify your destination is intact before enabling `--delete-after`.

## What's "Recently Deleted" got to do with this?

See [Free up iPhone storage](./guide/free-storage) — short version: DumpSock's delete frees the bytes immediately, but Photos.app keeps stale database rows that you have to clear with Face ID. We can't bypass that gate over USB.

## Does it leak any data?

No. DumpSock has zero network code outside Google Fonts (for the GUI's typography only, loaded from the CDN at app open). No analytics SDK, no crash reporter, no auto-update check. The source is open at [code-hartle-tech/dumpsock](https://github.com/code-hartle-tech/dumpsock) and the binaries are reproducibly built from tagged commits.

## Where's the code?

[github.com/code-hartle-tech/dumpsock](https://github.com/code-hartle-tech/dumpsock).

## Who makes it?

A [HARTLE.TECH](https://hartle.tech) tool. Email `contact@hartle.tech`.

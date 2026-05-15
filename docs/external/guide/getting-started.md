# What is DumpSock?

DumpSock is a **local-first** tool that pulls photos and videos off a USB-connected iPhone or iPad and writes them to a folder on your computer — or any external drive you plug in. Optionally, it deletes them from the phone afterward to free up storage.

There is no iCloud round-trip, no account, no signup, no telemetry. The bytes travel from your phone to your disk over the USB cable and nowhere else.

## Who it's for

- You shoot a lot of photos / video and your iPhone storage is full.
- You don't want to pay for iCloud, OR you do pay for iCloud but want a local archive too.
- You don't trust cloud-only backups (cloud accounts get locked, providers go bust, prices climb).
- You're a professional dumping camera roll mid-shoot to an SSD.
- You're a developer / power user who wants `cron` over their backups.

## What it's not

- **Not a cloud sync.** DumpSock has no cloud component. Your photos do not leave your machines.
- **Not a phone replacement.** It backs up the camera roll (`/var/mobile/Media/DCIM/`). It does not back up messages, settings, app data — for that, use Apple's encrypted backup via Finder.
- **Not an iOS app.** DumpSock runs on your Mac (and soon Linux / Windows). The iPhone just gets plugged in.

## How it compares

| Tool | Surface | Bytes route through |
|---|---|---|
| **DumpSock** | macOS GUI + CLI | iPhone → USB → your Mac. **Done.** |
| iCloud Photos | iOS Settings | iPhone → iCloud → optionally your Mac. Apple holds the keys (unless you also enable Advanced Data Protection). |
| `icloudpd` | CLI | iPhone → iCloud → your Mac. Cloud is the source of truth. |
| iMazing / AnyTrans | GUI | iPhone → USB → your Mac. Same idea as DumpSock, paid, closed-source. |
| `idevicebackup2` | CLI | iPhone → USB → your Mac. Full encrypted backup blob (everything), not a folder of files. |

## What's next

- [Install](./install) — three minutes on macOS.
- [Your first backup](./your-first-backup) — plug in, click Back Up Now, done.
- [Free up iPhone storage](./free-storage) — the `--delete-after` story and the Photos.app caveat.

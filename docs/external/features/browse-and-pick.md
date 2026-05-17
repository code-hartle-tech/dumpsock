# Browse what's on your iPhone — pick exactly what to back up

DumpSock can show you the contents of your iPhone before you copy anything. Tick the files or folders you actually want; leave the rest. No more all-or-nothing.

![Sock mascot looking through a magnifying glass](/dumpsock-mascot.svg){width=120}

## When to use Browse

- You only want **last weekend's videos**, not the whole camera roll.
- You're freeing space and want to confirm **what's actually big** before pulling.
- You want to grab files from a **third-party app's Documents folder** (DJI Fly flight logs, Procreate sketches, VLC media, Documents-by-Readdle archives).
- You want to **delete specific files from the phone** without going through Photos.app.

## How it works

DumpSock speaks Apple's **AFC** protocol — the same channel Finder uses when you click your iPhone in the macOS sidebar. That's it. No jailbreak, no proxy, no extra software on the phone.

For third-party apps, it speaks the **house_arrest** service to apps that ship `UIFileSharingEnabled=YES` in their `Info.plist`. That's the App-Store-compatible path — the one Apple intends for "expose this app's documents to Finder."

## What you'll see

Inside DumpSock's **Browse iPhone** tab:

- **Device source** — the iPhone's media partition. Top-level folders include `/DCIM` (photos + videos, by far the most useful), `/Books`, `/Recordings`, and a handful of system folders.
- **App data source** — a dropdown of every installed app that publishes its Documents folder via UIFileSharingEnabled. Pick one, browse its Documents tree.

Each row has a checkbox, a Delete button, and an icon indicating whether it's a file or a folder. Tick a folder and DumpSock will back up everything inside it. Tick a single file and you get only that file.

### What you WON'T see

iOS aggressively walls off areas you don't have permission to read over AFC:

- **App sandboxes for apps without `UIFileSharingEnabled`** — banking, messaging, social, system apps. Apple's review guidelines actively discourage exposing those.
- **`/PhotoData/Metadata`, `/PhotoData/Photos.sqlite`** since iOS 15+.
- **Keychain, Health, CloudKit** — none of these are on the AFC tree at all.

When DumpSock hits one of these subtrees, the row appears **greyed out with a 🔒 icon** and the size column reads *"hidden by iOS"* — so you know the folder isn't actually empty, you just can't get in over USB.

## Backing up the selection

After ticking files and folders, the footer shows your count and (for files) the byte total. Folders expand at backup time — DumpSock walks the folder and queues every file inside it.

Hit **Back up selected**. The summary toast tells you how many files the selection expanded to, then the backup starts on the **Backups** tab with the usual progress bar.

The selection is one-shot — pulling only the chosen files. The everything-pull from the Dashboard is unaffected and remains the default flow for full-camera-roll backups.

## Deleting from the phone

Each row also has a red **Delete** button. Click it, confirm the modal, and DumpSock issues an AFC remove for files or a recursive remove for folders. After the delete, DumpSock re-reads the path; if iOS says "deleted" but the file is still there (which happens for Photos.app-managed entries — they have ghost database records), the toast tells you honestly:

> *iOS held onto /DCIM/100APPLE/IMG_xxxx.HEIC (AFC reported success but path still exists — common on /PhotoData subtrees and Photos.app-managed entries; deletion from inside Photos is the workaround).*

For routine "free up space after a backup" workflows, prefer the Dashboard's **Wring it dry** option — it pulls *and* deletes in one pass, with verified-write safety.

## Caveats

- AFC's **rename** opcode isn't yet exposed by the Go library we use; the Rename button is disabled with a tooltip until that ships. Add a [comment on the upstream issue](https://github.com/danielpaulus/go-ios) if you want to nudge it forward.
- Pulling a single file through Browse doesn't write a backup-session checkpoint — if you Ctrl-Q mid-pull you lose progress on that file (whereas the Dashboard's everything-pull resumes cleanly).

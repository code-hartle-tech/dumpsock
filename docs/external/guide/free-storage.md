# Free up iPhone storage

DumpSock's optional **"wring it dry"** mode removes each file from the iPhone after it's safely on your disk. The bytes are freed immediately.

## How to enable

**GUI:** Settings → tick **"Wring it dry — delete from device after backup"** → run a backup → confirm in the modal that pops up.

**CLI:**
```bash
dumpsock pull --delete-after --confirm-delete
```

Both flags are required together — the CLI errors out if you set `--delete-after` without `--confirm-delete`. That's deliberate. The GUI has the same gate via the confirmation modal.

## How it actually works

DumpSock uses Apple's standard AFC (Apple File Conduit) service to talk to the iPhone over USB. After every file is verified at the destination (size + byte-count match the remote), DumpSock issues an AFC delete on the corresponding `/var/mobile/Media/DCIM/...` path. The bytes are gone from the filesystem **immediately** — iPhone storage drops by the size of the freed files.

## The Photos.app caveat

iOS keeps two layers between you and the photo bytes:

1. **The filesystem** at `/var/mobile/Media/DCIM/`. This is what DumpSock can reach over AFC.
2. **The Photos.app database** at `/var/mobile/Media/PhotoData/Photos.sqlite`. This is what the Photos app shows you. DumpSock cannot touch it (Apple sandboxes it).

When DumpSock removes a file from the DCIM filesystem, the Photos.app database still holds a row pointing to where that file *used to* live. So you'll see "ghost" entries in Photos with a small **!** badge and "unable to load higher quality version".

**This is normal.** The bytes are gone (your iPhone storage reflects it). The ghosts are just stale database rows.

## Clearing the ghosts

Apple intentionally requires you to confirm photo deletions with Face ID / Touch ID on the device. No USB tool can bypass that. Two-step manual cleanup:

1. Open **Photos** → **Library** → **All Photos**.
2. Tap any ghost → **Select** → **Select All** → trash icon → Confirm with Face ID.
3. Open **Albums** → **Recently Deleted** → **Select** → **Delete All** → Confirm with Face ID.

After step 3, the database rows are truly gone.

**One-Touch-ID automation is on the roadmap.** [Issue #1](https://github.com/code-hartle-tech/dumpsock/issues/1) tracks an iPhone Mirroring-based flow that automates the entire cleanup with a single Touch ID on the Mac — Apple's iPhone Mirroring intentionally substitutes Mac Touch ID for iPhone Face ID, which is the loophole.

## Why iPhone storage may take a few minutes to fully reflect

iOS lazily reindexes Photo storage. Right after a large delete-after run, the **Settings → General → iPhone Storage** number may take a few minutes to catch up to reality. Storage is freed at the filesystem layer instantly; the Settings number is just a cached aggregate.

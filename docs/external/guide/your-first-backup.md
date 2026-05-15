# Your first backup

## GUI (macOS)

1. Open **DumpSock.app**. The window opens maximised.
2. Plug in your iPhone over USB. The **Connected Device** card on the Dashboard fills in within a couple seconds — name, model, iOS version, free / total storage.
3. The **Storage Overview** donut shows what's eating space on your iPhone.
4. By default DumpSock saves into `~/DumpSock/<your-phone-name>/`. If you'd rather it land somewhere else (a fast external SSD, for example), click **Change…** under "Saves to" and pick a folder. The choice sticks for next time.
5. Click the big red **Back Up Now** button.
6. The view switches to **Backup Progress**. Watch the progress bar, the per-file stats, and the live filename. The first run of a phone with hundreds of photos can take a while — sit tight.
7. When it's done, you get a notification on the Mac and the screen shows **Done** with how many files moved.
8. Click **Open the folder** to see the date-sorted result in Finder.

## CLI

```bash
dumpsock pull
```

That's the common case. The defaults pull into `~/DumpSock/<device-name>/` with date folders. Useful flags:

```bash
dumpsock pull -o /Volumes/SSD/iPhone           # explicit destination
dumpsock pull --since 2026-01-01               # only files from Jan 1 2026 onward
dumpsock pull --dry-run                        # plan only, don't transfer
dumpsock pull --watch 60                       # rescan every 60s and pull anything new
dumpsock pull --help                           # full flag list
```

## What ends up on disk

```
~/DumpSock/iPhone/
├── 2026-05-15/
│   ├── IMG_0107.HEIC
│   ├── IMG_0107.MOV       ← Live Photo pair, same date folder
│   └── IMG_0108.MOV
├── 2026-05-14/
│   └── …
└── 0000:00:00 00:00:00/   ← files with no readable capture date
    └── …
```

The `YYYY-MM-DD` layout is the same one [icloudpd](https://github.com/icloud-photos-downloader/icloud_photos_downloader) uses — if you have an existing icloudpd backup, DumpSock can fill the same tree without duplicates (it skips files where the name + byte size already exist somewhere in the destination).

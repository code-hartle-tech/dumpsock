# File layout on disk

DumpSock writes a flat tree of `YYYY-MM-DD/` folders under your chosen destination root:

```
<destination root>/
├── 2025-08-12/
│   ├── IMG_4321.HEIC
│   ├── IMG_4321.MOV
│   ├── IMG_4321.AAE             # if you'd cropped or filtered it
│   ├── IMG_4322-1.HEIC          # different content, same filename → -1 suffix
│   └── BURST_4323_COVER.HEIC
├── 2025-08-13/
│   └── …
└── 0000:00:00 00:00:00/         # files with no readable date metadata
    └── …
```

## File naming

Filenames are preserved verbatim from the iPhone — DumpSock does **not** rename anything. The iPhone's DCIM uses:

| Pattern | What |
|---|---|
| `IMG_NNNN.HEIC` | High-Efficiency Image Format photo |
| `IMG_NNNN.JPG` / `.JPEG` | JPEG photo (older iPhones, screenshots in some configurations) |
| `IMG_NNNN.PNG` | PNG (mostly screenshots) |
| `IMG_NNNN.MOV` | Video, or the Live Photo motion component |
| `IMG_NNNN.MP4` | Compressed video shared from elsewhere |
| `IMG_NNNN.DNG` | Apple ProRAW |
| `IMG_NNNN.AAE` | Apple Adjustments Edit metadata sidecar |
| `IMG_E…` / `IMG_O…` | Edited / Original variants Apple keeps when you edit a Live Photo |
| `BURST_…` | Burst-mode cover + sequence |

## What lives where on the iPhone

For the curious — this is what DumpSock walks on the device:

```
/var/mobile/Media/DCIM/
├── 100APPLE/    ← Apple's standard DCIM camera folder
├── 101APPLE/    ← rolls over every 999 photos
├── …
├── 110APPLE/    ← current at time of writing for an iPhone 16 Pro Max
└── .MISC/       ← burst metadata, timelapse intermediates (ignored)
```

DumpSock walks all `*APPLE/` subdirectories recursively, picks up media files by extension, and skips the `.MISC/` housekeeping tree.

## Excluded by default

- Anything starting with `.` (hidden files)
- Anything not in the media extension allow-list: `.heic .heif .jpg .jpeg .png .mov .mp4 .m4v .dng .raw .gif .webp`
- `.AAE` sidecars are NOT pulled by default — they're metadata, not media. They ARE deleted alongside their parent file when `--delete-after` is set.

Override the allow-list / scope with `--remote-root PATH` (default `DCIM`) — DumpSock can walk anywhere AFC exposes.

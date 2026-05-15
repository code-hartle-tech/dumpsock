# exiftool `-fast2` trap

## TL;DR

`exiftool -fast2` reads the EXIF block at the *start* of a file and skips deep-structure atoms. For MOV/MP4 videos, the capture date lives in the QuickTime **`moov`** atom, which is often **at the end** of the file. `-fast2` skips it. exiftool then returns no date, the caller falls back to file mtime, every video gets dated to "today". Use `-fast` (which reads moov) or, better, replace exiftool entirely.

## What it looked like

The user ran a backup at noon today. The destination had:

```
2026-05-15/
├── IMG_4321.HEIC      ← correct
├── IMG_4321.MOV       ← WRONG — should be in 2025-08-12/
└── IMG_5012.MOV       ← WRONG — should be in 2025-03-04/
```

All HEIC photos were in the right date folders. All MOV files landed under today's date.

## Why

We were invoking exiftool as:

```bash
exiftool -fast2 -CreateDate -DateTimeOriginal -s -s -s "$file"
```

For HEIC, the EXIF block is in the first ~10 KB of the file. `-fast2` reads it. Date found. Right folder.

For MOV/MP4, the QuickTime container's `moov` atom — which has the `CreateDate` we need — is often **at the end** of the file because the encoder writes the `mdat` (data) atom first and the `moov` last (this is normal; MOV doesn't require moov to be at the start). `-fast2` doesn't seek to the end. No date found. exiftool returns empty.

Our caller saw the empty result, fell back to `FileModifyDate`, which the iPhone had updated to "the time the file was last accessed via AFC" — i.e., the moment of the backup. Today.

## The fix in the old code path

```diff
- exiftool -fast2 -CreateDate -DateTimeOriginal -s -s -s "$file"
+ exiftool -fast  -CreateDate -DateTimeOriginal -MediaCreateDate -s -s -s "$file"
```

Two changes:
1. `-fast2` → `-fast`. `-fast` still skips the metadata-tail re-read at the end, but DOES walk the moov atom.
2. Add `-MediaCreateDate` to the precedence chain. Older MOVs (and some Android-encoded MP4s) use `MediaCreateDate` not `CreateDate`.

## The fix in the current code path

We rewrote the EXIF reader as pure Go. exiftool is no longer a dependency.

```go
// internal/exif/exif.go
func Read(path string) (time.Time, error) {
    // 1. Try imagemeta — handles HEIC, JPEG, PNG, raw via standard EXIF.
    if t, err := readWithImagemeta(path); err == nil && !t.IsZero() {
        return t, nil
    }
    // 2. Fall back to our moov walker for MOV/MP4/M4V.
    if t, err := walkMoov(path); err == nil && !t.IsZero() {
        return t, nil
    }
    // 3. Last resort: filesystem mtime.
    return statMTime(path)
}
```

`walkMoov` is a custom ISO-BMFF reader that:
1. Reads the first 8 bytes of each top-level atom.
2. Hops by atom size.
3. Stops when it hits `moov`.
4. Walks into `moov.mvhd` (movie-header).
5. Reads `creation_time` (a 32 or 64-bit count of seconds since 1904-01-01 UTC).
6. Converts to `time.Time`.

It works for the moov-at-end case because we explicitly seek and read forward atom-by-atom.

## Why we ripped out exiftool

- **Speed.** exiftool is Perl, fork-per-file. For ~10,000 files, that's ~30 seconds of forking. Pure-Go EXIF is ~150 ms total.
- **Dependency.** Requiring exiftool installed limits us to users who can `brew install exiftool`. We want a single static binary.
- **Trap surface.** Future Claude sessions might flip back to `-fast2` not knowing why we explicitly chose `-fast`. With pure-Go code, the precedence chain is right there in the source.

## How we recovered

The user already had ~500 misplaced MOV files. Re-pulling 70 GB was not appealing. So we wrote [`dumpsock resort`](../runbooks/resort-misplaced), which walks the destination, re-reads each file's EXIF/moov with the current reader, and moves files to the correct date folder.

## Code

- `internal/exif/exif.go` — the pure-Go reader + moov walker.
- `internal/exif/moov.go` — the ISO-BMFF atom walker.
- `internal/exif/exif_test.go` — table-driven against `testdata/` fixtures.
- `cmd/dumpsock/cmd/resort.go` — the recovery subcommand.

## Mental model

> Capture date for video lives in the moov atom. The moov atom can be anywhere in the file. Any tool that promises "fast" EXIF reads on video without walking moov is lying. Walk moov, or accept that video dates are wrong.

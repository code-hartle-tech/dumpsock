# CLI flags reference

Full list of `dumpsock pull` flags, including the ones hidden from default `--help`.

```
dumpsock pull [flags]
```

## Visible flags (default `--help`)

| Flag | Default | What |
|---|---|---|
| `-o`, `--output DIR` | `~/DumpSock/<device-name>` | Destination folder |
| `--since YYYY-MM-DD` | (none) | Only pull files captured on/after |
| `--until YYYY-MM-DD` | (none) | Only pull files captured on/before |
| `--delete-after` | false | Remove each file from device after verified copy (requires `--confirm-delete`) |
| `--confirm-delete` | false | Explicit confirmation gate for `--delete-after` |
| `--watch N` | 0 | Stay running, rescan every N seconds |
| `--dry-run` | false | Plan only — list what would be pulled, transfer nothing |
| `--udid UDID` | (auto) | Pick specific device when multiple connected |
| `-h`, `--help` | — | Show help |

## Advanced flags (hidden, see `--help-advanced` or this page)

| Flag | Default | What |
|---|---|---|
| `--parallel N` | 4 | Concurrent post-pull workers (EXIF read + move) |
| `--until-found N` | 0 | Stop scan after N consecutive name+size matches against existing destination |
| `--no-mtime` | false | Don't set file mtime to capture date |
| `--no-live-pair` | false | Don't co-locate HEIC + MOV Live Photo pairs |
| `--no-notify` | false | No desktop notification on completion |
| `--hash size\|sha256` | size | Dedup mode |
| `--remote-root PATH` | DCIM | Path on the device to walk |
| `--json` | false | Machine-readable progress stream on stdout |

## Other subcommands

```
dumpsock devices [--json]      # list connected iOS devices
dumpsock version               # version + build info
dumpsock --help
```

## Environment

- `usbmuxd` must be running locally.
- `exiftool` is **not** required — DumpSock uses a pure-Go EXIF reader (`imagemeta`) + a built-in QuickTime moov-atom walker.

## What gets printed

```
indexing existing files under <output>
indexed <N> existing files
walking remote DCIM/ ...
remote media files: <M> (+ <S> sidecars)
to pull: <K> (pre-skipped: <P>)
  N/K  pulled=… skipped=… suffixed=… nodate=… errors=…  (rate)
done: total=… pre_skipped=… pulled=… post_skipped=… suffixed=… nodate=… filtered=… errors=… [deleted=… delete_errors=…] (elapsed)
```

With `--json`, the same information is emitted as one JSON object per progress event on stdout, suitable for piping into `jq` or a TUI.

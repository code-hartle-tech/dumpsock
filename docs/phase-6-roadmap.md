# Phase 6 / 7 roadmap — Cloud + Full-Device Backup

> Scope-expansion plan for adding iCloud Photos download and full-device backup to DumpSock. Drafted 2026-05-17 from a parallel-agent research pass on the iCloud auth landscape, MobileBackup2 protocol shape, and go-ios's current support status. Implementation is staged behind opt-in subcommands so the default `dumpsock` flow stays local-AFC-first.

## TL;DR

| Phase | Subcommand | Status (2026-05-17) | Effort |
|---|---|---|---|
| 6 — Full-device backup | `dumpsock backup-device` | Scaffolded (`internal/mb2/`) | L–XL · 2–10 weeks |
| 7 — iCloud Photos pull | `dumpsock icloud pull` | Scaffolded (`internal/icloud/`) | L · 2–3 weeks + ongoing maintenance tax |
| 8 — Backup extraction | `dumpsock backup extract` | Future | M · ~1 week (depends on Phase 6) |
| Restore-to-device | — | **Out of scope** (broken on iOS 17/18 for third-party tools) | — |

## Phase 6 — MobileBackup2 backup engine

**Goal:** produce a Finder/iTunes-equivalent backup tree at `~/DumpSock/<device>/backup/<timestamp>/` (Manifest.db, Manifest.plist, Info.plist, Status.plist, hashed-blob storage under `00/01/.../ff/`), without shelling out to `idevicebackup2` (CLAUDE.md "no external binary" rule).

### Protocol surface (per agent research)

Lockdownd services involved:
- `com.apple.mobilebackup2` — the backup channel itself (DeviceLink-framed plist messages over lockdown TLS)
- `com.apple.mobile.installation_proxy` — enumerates installed apps for the manifest
- `com.apple.mobile.notification_proxy` — sync-cancel / domain-change notifications
- `com.apple.mobile.afc` — writes `iTunesPrefs`/`Status.plist` markers on the device's media partition
- `com.apple.mobile.diagnostics_relay` — power-state / on-AC checks
- `com.apple.springboard.sbservices` — app icons for the manifest

Message flow: lockdown handshake → start `mobilebackup2` → `Hello` + `SupportedProtocolVersions` (2.0/2.1) → client sends `BackupRequest` with `WillEncrypt`/`MightHaveEnabledCloudBackup` → device drives a DeviceLink loop of `DLMessage*` frames (`DownloadFiles`, `UploadFiles`, `GetFreeDiskSpace`, `ContentsOfDirectory`, `CreateDirectory`, `MoveFiles`, `RemoveFiles`, `CopyItem`, `Disconnect`, `ProcessMessage`) with length-prefixed plist tuples carrying raw bytes.

Encryption is negotiated **out-of-band** — the password lives in iOS Settings > General > Reset > "Encrypt iPhone Backup". Wire only carries a `WillEncrypt` bool. `BackupAgent2` on-device handles AES + key wrapping; the host never sees plaintext keys.

### On-disk format (per agent research)

- **Manifest.db (SQLite)** — single `Files` table: `fileID` (SHA-1 of `domain + "-" + relativePath`), `domain`, `relativePath`, `flags`, `file` (NSKeyedArchiver "MBFile" plist with size/mtime/mode/EncryptionKey).
- **Info.plist** — device metadata (Display Name, Product Type, Serial, IMEI, installed apps). Cleartext even when encrypted.
- **Status.plist** — `IsFullBackup`, `SnapshotState`, `Date`, `UUID`, `Version`.
- **Manifest.plist** — `Version`, `Date`, `Lockdown` dict, `Applications` dict; on encrypted backups adds `BackupKeyBag` (binary TLV keybag) + `ManifestKey` (wrapped AES key for Manifest.db).
- **Storage layout** — each file's payload at `<UDID>/<first-2-hex>/<full-40-hex-fileID>`. No extensions, no folder mirroring.

### go-ios status

`v1.0.213` has **zero** MobileBackup2 support — single string mention in a test fixture, no package. Reusable from go-ios: `ios/afc`, `ios/notificationproxy`, `ios/installationproxy`, `ios/diagnostics`, `ios/springboard`, `ios/nskeyedarchiver`, plist machinery.

### Porting plan

1. **DeviceLink framing layer** (~few hundred LOC): length-prefixed plist tuples, the envelope every mb2 frame rides in.
2. **`DLMessage*` dispatcher** (~1k LOC): every verb must be exact or the device aborts. Status-plist replies for each.
3. **Manifest writer** — produce byte-for-byte iTunes-compatible Manifest.plist/Status.plist/Info.plist.
4. **Manifest.db writer** — `modernc.org/sqlite` (CGo-free).
5. **Lockdown wiring** — escrow-bag / pair-record reuse for the `mobilebackup2` service start.
6. **Progress + cancel** — via `notification_proxy` for the `com.apple.itunes-mobdev.syncCancelRequest` event.

**Reference implementations to cross-check:** `libimobiledevice/tools/idevicebackup2.c`, `libimobiledevice/src/device_link_service.c`, `dunhamsteve/ios` (Go), `avibrazil/iOSbackup` (Python), `as0ler/iphone-dataprotection`.

**Realistic effort:** 2–4 weeks for read-only unencrypted backup MVP. 6–10 weeks for `idevicebackup2 backup --full` parity (incl. encrypted toggle).

**MVP shortcut:** ship **unencrypted backup-only** first. Skips the password ChangePassword dance entirely. Encrypted-toggle is a Settings switch on the device; wire code is identical.

### Scaffold location

- `internal/mb2/mb2.go` — package doc, `Options`, `Backup(opts)`, `ErrNotImplemented`.
- `internal/cli/backupdevice.go` — `dumpsock backup-device` Cobra command.

## Phase 7 — iCloud Photos download

**Goal:** read-only `dumpsock icloud pull` that mirrors `icloudpd`'s UX — pulls photos/videos from iCloud Photos into the same YYYY-MM-DD/ tree the local-AFC `dumpsock pull` produces, so the two outputs are interchangeable.

### Auth landscape (Q1 2026, per agent research)

- App-specific passwords have **never worked** for iCloud Photos (Mail/Calendar/Contacts/Reminders only).
- icloudpd switched to Apple's **SRP-6a** challenge protocol in mid-2024. Plain password POSTs to `idmsa.apple.com/appleauth/auth/signin` now return 503/421.
- ADP-enabled accounts cannot be served — Apple disabled web access there.

Flow:
1. SRP-6a init (`/signin/init`) → server salt + B; client computes M1.
2. `/signin/complete` with M1 → `X-Apple-Session-Token`.
3. HSA2 2FA: code from trusted device → `/verify/trusteddevice/securitycode`.
4. `/2sv/trust` → long-lived `X-APPLE-WEBAUTH-HSA-TRUST` cookie (~30 days).
5. `setup.icloud.com/setup/ws/1/accountLogin` → session cookies + per-account `dsid` and zone routing (`pNN-ckdatabasews.icloud.com`).

### Transport

- Setup: `setup.icloud.com`
- CloudKit Web Services: `pNN-ckdatabasews.icloud.com/database/1/com.apple.photos.cloud/production/private/records/query` with zone `PrimarySync` (asset records) and `SharedSync-*` (shared library).
- Per-asset content: `pNN-content.icloud.com`.

### Risk register

- **Concurrent-request limit** — Apple IP-blacklists at >6 concurrent POSTs/IP → 503s. Throttle on the client.
- **Apple breakage cadence** — icloudpd has shipped emergency releases roughly every 6–9 months since 2024. Expect a maintenance release every ~2 quarters.
- **Re-auth storms** — watch-mode token expiry routinely triggers 421s.

### Go library landscape

No production-grade pure-Go iCloud client exists. Surveyed: `chyroc/icloudgo` (Dec 2023, pre-SRP — won't auth today), `lukasmalkmus/icloud-go` (2021, official CloudKit-Web only), `alvaroaleman/icloud-go` (dead). Realistic path: port `pyicloud_ipd` (~3.5kLOC) to Go. SRP needs `github.com/1Password/srp` or `tadglines/go-pkgs/crypto/srp`.

**Effort:** 2–3 focused weeks for parity with `--since/--until/--live`.

### Scaffold location

- `internal/icloud/icloud.go` — package doc, `Options`, `Pull(opts)`, `ErrNotImplemented`.
- `internal/cli/icloud.go` — `dumpsock icloud pull` Cobra command.

## Phase 8 — Backup extraction (not restore)

Restore-to-device is **out of scope**. Even libimobiledevice's restore is unreliable on iOS 17 and broken on 18 (open issues #1548, #1283, #1681). Activation Lock + encryption-password UX would be a support nightmare.

What *is* feasible and useful for users:

| Command | Effort | Notes |
|---|---|---|
| `dumpsock backup verify <path>` | M | Manifest.db integrity + per-file SHA verification |
| `dumpsock backup extract <path> --filter Photos` | M | Pull files OUT of a Manifest.db tree into a normal folder — what most "iPhone backup extractor" tools actually do |
| `dumpsock backup mount <path>` | L | FUSE-mount via macFUSE / WinFsp; needs CGo, ship behind a `fuse` build tag (CLAUDE.md "extended binary") |

Hard-drop "restore-to-device" entirely. Reframe Phase 8 as **backup inspection & extraction**.

## Cross-cutting

- **Single-binary promise** (CLAUDE.md line 23) survives Phase 6 (pure-Go mb2 port) and Phase 7 (pure-Go iCloud); breaks on FUSE-mount → behind build tag.
- **Encrypted-backup password** and **iCloud trust-token** both want OS-keyring storage. Design `internal/keyring/` once (`zalando/go-keyring`), use twice.
- **`internal/backup` rename** — currently "backup" means "photo pull". Before Phase 6 lands, rename to `internal/pull/` to free the name for the mb2 engine. Tracked separately.

## Open decisions

- Whether Phase 6 ships unencrypted-only first (recommended MVP) or waits for the full encrypted-toggle implementation.
- Whether Phase 7 uses a vendored fork of `chyroc/icloudgo` (after porting SRP onto it) or starts fresh from `pyicloud_ipd`.
- Whether to expose Phase 6/7 in the GUI at v1 or hold them CLI-only until they stabilize.

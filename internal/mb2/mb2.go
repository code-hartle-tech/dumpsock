// Package mb2 is the (scaffold) MobileBackup2 client. Phase 6 of the
// DumpSock roadmap. Today every entry point returns ErrNotImplemented.
// The protocol shape, services used, and porting plan live in
// docs/phase-6-roadmap.md.
//
// Protocol summary: lockdownd starts com.apple.mobilebackup2; the
// client speaks DeviceLink-framed plist messages. Backup flow is
// BackupRequest → device-driven DownloadFiles/UploadFiles loop →
// final commit. Encrypted-backup password is established in iOS
// Settings (not via the protocol); the wire only carries a WillEncrypt
// flag. Restore is symmetric in services but iOS-version-fragile and
// currently broken for third-party tools on iOS 17/18 (see agent
// report in session 2026-05-17).
//
// go-ios status: no MobileBackup2 package in v1.0.213. A pure-Go port
// of libimobiledevice's idevicebackup2.c + device_link_service.c is
// the largest single workitem in the project (~3-5kLOC).
package mb2

import "errors"

// ErrNotImplemented marks every scaffold entry point until Phase 6
// lands.
var ErrNotImplemented = errors.New("mb2: not yet implemented (Phase 6 in development; see docs/phase-6-roadmap.md)")

// Options will configure a Backup() call once the protocol port lands.
type Options struct {
	UDID       string
	OutputRoot string
	WillEncrypt bool
	// Password is supplied on the wire only when restoring; for backup
	// the encryption toggle is on-device.
	Password string
}

// Backup will run a full-device backup into OutputRoot. Today it just
// returns ErrNotImplemented.
func Backup(opts Options) error {
	_ = opts
	return ErrNotImplemented
}

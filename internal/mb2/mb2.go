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

import (
	"errors"
	"fmt"

	"github.com/danielpaulus/go-ios/ios"
)

// ErrNotImplemented marks scaffold entry points that don't have a
// working implementation yet. Backup() still returns this — the
// dispatcher loop + manifest writers (layers 2 + 3) are pending.
// Probe() does NOT return this anymore; it does real protocol work.
var ErrNotImplemented = errors.New("mb2: not yet implemented (see docs/phase-6-roadmap.md)")

// Options will configure a Backup() call once the protocol port lands.
type Options struct {
	UDID        string
	OutputRoot  string
	WillEncrypt bool
	// Password is supplied on the wire only when restoring; for backup
	// the encryption toggle is on-device.
	Password string
}

// Backup will run a full-device backup into OutputRoot. Today it just
// returns ErrNotImplemented — layers 2 (dispatcher) + 3 (manifest
// writers) are pending. Probe() below DOES work; use it to confirm
// the path is open.
func Backup(opts Options) error {
	_ = opts
	return ErrNotImplemented
}

// Probe opens the com.apple.mobilebackup2 lockdown service and runs
// the DeviceLink version-exchange handshake, returning the negotiated
// protocol version on success. Touches nothing on the device — purely
// confirms the wire path works before we invest in the full backup
// flow.
//
// This is Phase 6's Layer 1 deliverable. The `dumpsock backup-device
// --probe` CLI flag calls it; future GUI status panels can too.
func Probe(udid string) (Handshake, error) {
	dev, err := pickDevice(udid)
	if err != nil {
		return Handshake{}, err
	}
	conn, err := openService(dev)
	if err != nil {
		return Handshake{}, err
	}
	defer conn.Close()
	return versionExchange(conn)
}

// pickDevice resolves a UDID to a go-ios DeviceEntry. Empty UDID picks
// the only attached device. Duplicated here (vs internal/gui) to keep
// package boundaries clean — mb2 is meant to be importable from CLI
// without dragging in the GUI's transitive deps.
func pickDevice(udid string) (ios.DeviceEntry, error) {
	list, err := ios.ListDevices()
	if err != nil {
		return ios.DeviceEntry{}, fmt.Errorf("usbmuxd: %w", err)
	}
	if len(list.DeviceList) == 0 {
		return ios.DeviceEntry{}, errors.New("no iPhone connected over USB")
	}
	if udid == "" {
		if len(list.DeviceList) > 1 {
			return ios.DeviceEntry{}, errors.New("multiple devices — pass --udid")
		}
		return list.DeviceList[0], nil
	}
	for _, d := range list.DeviceList {
		if d.Properties.SerialNumber == udid {
			return d, nil
		}
	}
	return ios.DeviceEntry{}, fmt.Errorf("device %s not connected", udid)
}

package mb2

import (
	"fmt"

	"github.com/danielpaulus/go-ios/ios"
)

const (
	// mobileBackup2ServiceName is the lockdownd service that hosts
	// every interaction with iOS's full-device backup engine. Apple
	// has kept this name stable since iOS 7.
	mobileBackup2ServiceName = "com.apple.mobilebackup2"

	// supportedProtocolMajor / Minor are the MobileBackup2 wire-protocol
	// version we advertise. Apple's scheme uses big integers (the iOS
	// device offers "400.0" in 2026-05); libimobiledevice's idevicebackup2
	// matches whatever the device offers. We do the same — cap at our
	// known-good ceiling but acknowledge any equal-or-lower version the
	// device proposes.
	supportedProtocolMajor = 400
	supportedProtocolMinor = 0
)

// Handshake is the result of a successful mb2 protocol negotiation.
// Surface via Probe so callers (CLI --probe flag, GUI status panel)
// can confirm the path is open before kicking off a real backup.
type Handshake struct {
	// NegotiatedMajor / NegotiatedMinor are the version numbers the
	// device chose. Returned in the device's DLMessageVersionExchange
	// reply (third array element).
	NegotiatedMajor int
	NegotiatedMinor int

	// ProtocolReady is true once the device-ready exchange completes
	// (the device sends DLMessageDeviceReady after the version match).
	ProtocolReady bool

	// RawOffer captures the device's initial frame exactly as we
	// received it (after plist→Go decoding). Surfaced in error
	// messages so we can iterate the frame-shape assumptions if
	// libimobiledevice's docs and Apple's actual wire format disagree.
	RawOffer []interface{}
}

// openService opens the lockdown service for mobilebackup2 and wraps
// it in a DeviceLink-framing connection. Caller closes the returned
// dlConn.
func openService(dev ios.DeviceEntry) (*dlConn, error) {
	conn, err := ios.ConnectToService(dev, mobileBackup2ServiceName)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", mobileBackup2ServiceName, err)
	}
	return newDLConn(conn), nil
}

// versionExchange runs the MobileBackup2 handshake. Returns the
// negotiated protocol version on success.
//
// Wire flow (verified against a real iPhone running iOS 18 on
// 2026-05-18 — the device sent ["DLMessageVersionExchange", 400, 0]):
//
//  1. Device pushes DLMessageVersionExchange first (NOT host — mb2
//     inverts the usual host-drives-loop pattern for this one
//     handshake). Frame shape:
//        ["DLMessageVersionExchange", major, minor]
//     where major/minor are integers (Apple uses 400.0 currently).
//
//  2. Host replies with:
//        ["DLMessageVersionExchange", "DLVersionsOk", major]
//     "DLVersionsOk" is the literal string ack; major is the version
//     we agree to speak (typically equal to what the device offered,
//     capped at our known-good ceiling).
//
//  3. Device replies with ["DLMessageDeviceReady"]. After that the
//     dispatcher loop owns the channel.
func versionExchange(d *dlConn) (Handshake, error) {
	// Step 1: read the device's initial offer.
	offer, err := d.readFrame()
	if err != nil {
		return Handshake{}, fmt.Errorf("read VersionExchange offer: %w", err)
	}
	if frameName(offer) != "DLMessageVersionExchange" {
		return Handshake{}, fmt.Errorf("expected DLMessageVersionExchange, got %q", frameName(offer))
	}
	if len(offer) < 2 {
		return Handshake{RawOffer: offer}, fmt.Errorf("VersionExchange offer has no version (raw = %#v)", offer)
	}
	deviceMajor := asInt(offer[1])
	deviceMinor := 0
	if len(offer) >= 3 {
		deviceMinor = asInt(offer[2])
	}
	if deviceMajor == 0 {
		return Handshake{RawOffer: offer}, fmt.Errorf("VersionExchange major version not parseable (raw = %#v)", offer)
	}

	// Pick a version we BOTH support. We cap at our supported ceiling;
	// otherwise take whatever the device offered.
	useMajor, useMinor := deviceMajor, deviceMinor
	if useMajor > supportedProtocolMajor ||
		(useMajor == supportedProtocolMajor && useMinor > supportedProtocolMinor) {
		useMajor, useMinor = supportedProtocolMajor, supportedProtocolMinor
	}

	// Step 2: reply. Format is ["DLMessageVersionExchange", "DLVersionsOk", major].
	reply := []interface{}{
		"DLMessageVersionExchange",
		"DLVersionsOk",
		uint64(useMajor),
	}
	if err := d.sendFrame(reply); err != nil {
		return Handshake{}, fmt.Errorf("send VersionExchange reply: %w", err)
	}

	// Step 3: device confirms with DLMessageDeviceReady.
	ready, err := d.readFrame()
	if err != nil {
		return Handshake{}, fmt.Errorf("read DLMessageDeviceReady (device may have rejected our protocol-version reply): %w", err)
	}
	if frameName(ready) != "DLMessageDeviceReady" {
		return Handshake{RawOffer: offer}, fmt.Errorf("expected DLMessageDeviceReady, got %q (raw = %#v)", frameName(ready), ready)
	}

	return Handshake{
		NegotiatedMajor: useMajor,
		NegotiatedMinor: useMinor,
		ProtocolReady:   true,
		RawOffer:        offer,
	}, nil
}

// asInt narrows the various integer types plist decoding can hand us
// (uint64 most often; sometimes int64 or float64) into a Go int.
func asInt(v interface{}) int {
	switch x := v.(type) {
	case uint64:
		return int(x)
	case int64:
		return int(x)
	case int:
		return x
	case float64:
		return int(x)
	}
	return 0
}

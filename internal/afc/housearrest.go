// Package afc wraps go-ios's AFC client. This file adds a minimal
// HouseArrest client that supports BOTH the `VendContainer` message
// (whole app sandbox; only available on debuggable / dev-signed apps)
// AND `VendDocuments` (Documents/ only; the path the App Store policy
// blesses for production-signed apps with UIFileSharingEnabled=YES).
//
// go-ios v1.0.213's house_arrest.New hardcodes VendContainer, which
// makes it useless for the common case — production-signed apps like
// DJI GO Lite (com.dji.golite). That manifests in the GUI as
// "InstallationLookupFailed" on every real-world app the user tries.
// Until an upstream PR lands, we ship this wrapper inline.

package afc

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"howett.net/plist"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/afc"
)

const houseArrestService = "com.apple.mobile.house_arrest"

// houseArrestResponse mirrors the {Status, Error} plist the device
// sends back after a Vend* command.
type houseArrestResponse struct {
	Status string
	Error  string
}

// OpenAppContainer connects to com.apple.mobile.house_arrest, requests
// access to a third-party app's sandbox, and returns a DumpSock-flavored
// AFC Client rooted at whatever the vend command authorised.
//
// Strategy: try `VendDocuments` first (works for production-signed apps
// with UIFileSharingEnabled=YES — the realistic majority). If the
// device rejects with anything other than "Complete", retry on a fresh
// connection with `VendContainer` (works for dev-signed apps that allow
// the broader container access). If both fail, surface the second error
// (which is the more useful one for diagnosing).
func OpenAppContainer(udid, bundleID string) (*Client, error) {
	if bundleID == "" {
		return nil, errors.New("OpenAppContainer: empty bundleID")
	}
	dev, err := pickDevice(udid)
	if err != nil {
		return nil, err
	}
	if cl, err := openAppVend(dev, bundleID, "VendDocuments"); err == nil {
		name := ""
		if v, gerr := ios.GetValues(dev); gerr == nil {
			name = v.Value.DeviceName
		}
		return &Client{inner: cl, dev: dev, name: name}, nil
	} else {
		// Fall back to VendContainer. Helpful for dev-signed apps.
		cl, cerr := openAppVend(dev, bundleID, "VendContainer")
		if cerr != nil {
			return nil, fmt.Errorf("house_arrest %s: VendDocuments=%v; VendContainer=%v", bundleID, err, cerr)
		}
		name := ""
		if v, gerr := ios.GetValues(dev); gerr == nil {
			name = v.Value.DeviceName
		}
		return &Client{inner: cl, dev: dev, name: name}, nil
	}
}

func openAppVend(dev ios.DeviceEntry, bundleID, command string) (*afc.Client, error) {
	conn, err := ios.ConnectToService(dev, houseArrestService)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", houseArrestService, err)
	}
	codec := ios.NewPlistCodec()
	req := map[string]interface{}{"Command": command, "Identifier": bundleID}
	msg, err := codec.Encode(req)
	if err != nil {
		return nil, fmt.Errorf("encode %s plist: %w", command, err)
	}
	if err := conn.Send(msg); err != nil {
		return nil, fmt.Errorf("send %s: %w", command, err)
	}
	response, err := codec.Decode(conn.Reader())
	if err != nil {
		// EOF here means the device closed the socket without replying —
		// the documented signal Apple uses to refuse house_arrest access
		// to system apps (com.apple.*) and to apps that don't actually
		// participate in the service, even when they advertise
		// UIFileSharingEnabled=YES.
		emsg := err.Error()
		if strings.Contains(emsg, "EOF") || strings.Contains(emsg, "io: read") {
			return nil, fmt.Errorf("iOS refused to vend this app's files. This is normal for system apps (com.apple.*) and for third-party apps that don't actually enable file sharing — pick a different app, or check that the app appears under Finder → iPhone → Files")
		}
		return nil, fmt.Errorf("decode %s response: %w", command, err)
	}
	var resp houseArrestResponse
	if err := plist.NewDecoder(bytes.NewReader(response)).Decode(&resp); err != nil {
		return nil, fmt.Errorf("parse %s response plist: %w", command, err)
	}
	if resp.Status != "Complete" {
		if resp.Error != "" {
			return nil, fmt.Errorf("%s rejected by device: %s", command, resp.Error)
		}
		return nil, fmt.Errorf("%s rejected by device (no status detail)", command)
	}
	return afc.NewFromConn(conn), nil
}

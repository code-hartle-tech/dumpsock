// Package afc wraps go-ios's AFC client with the subset of operations
// DumpSock needs: device discovery, recursive walk of a remote directory,
// stat + stream-pull single files.
package afc

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/afc"
)

// File is one regular-file entry discovered by Walk.
type File struct {
	Path string // absolute remote path, slash-separated
	Name string // basename only
	Size int64  // bytes
}

// Client is a stateful AFC connection. Callers must Close it.
type Client struct {
	inner *afc.Client
	dev   ios.DeviceEntry
	name  string // device name, set after open if available
}

// Open dials a device by UDID (empty = pick the only USB-connected one)
// and starts an AFC session against it.
func Open(udid string) (*Client, error) {
	dev, err := pickDevice(udid)
	if err != nil {
		return nil, err
	}
	inner, err := afc.New(dev)
	if err != nil {
		return nil, fmt.Errorf("AFC handshake failed (is the device unlocked + trusted?): %w", err)
	}
	name := ""
	if v, err := ios.GetValues(dev); err == nil {
		name = v.Value.DeviceName
	}
	return &Client{inner: inner, dev: dev, name: name}, nil
}

// Close releases the AFC connection.
func (c *Client) Close() error {
	if c.inner == nil {
		return nil
	}
	return c.inner.Close()
}

// DeviceName returns the device's displayed name (e.g. "IoT"), or empty if
// it couldn't be read.
func (c *Client) DeviceName() string { return c.name }

// UDID returns the device UDID.
func (c *Client) UDID() string { return c.dev.Properties.SerialNumber }

// Walk recursively enumerates regular files under remoteRoot, filtered by
// extension (case-insensitive, leading dot, e.g. ".heic"). Returns the
// matched files in discovery order.
//
// Hidden entries (basename starting with ".") and "." / ".." are skipped.
func (c *Client) Walk(remoteRoot string, allowedExts map[string]bool) ([]File, error) {
	var out []File
	err := c.inner.WalkDir(remoteRoot, func(p string, info afc.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil // skip the entry, keep walking
		}
		if info.IsDir() {
			return nil
		}
		name := info.Name
		if name == "" {
			name = filepath.Base(p)
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if allowedExts != nil && !allowedExts[ext] {
			return nil
		}
		out = append(out, File{
			Path: p,
			Name: name,
			Size: info.Size,
		})
		return nil
	})
	return out, err
}

// PullTo copies a single remote file to localPath (which must include the
// destination filename). Stream-based; large files don't buffer in memory.
func (c *Client) PullTo(remotePath, localPath string) error {
	return c.inner.PullSingleFile(remotePath, localPath)
}

// Remove deletes a single regular file on the device. Use only after the
// local copy has been verified — there is no recycle bin on iOS.
func (c *Client) Remove(remotePath string) error {
	return c.inner.Remove(remotePath)
}

// Storage is the device's filesystem capacity snapshot as reported by
// AFC. TotalBytes is the iPhone's total storage; FreeBytes is currently
// available; Model is the iPhone marketing model where known.
type Storage struct {
	Model      string
	TotalBytes uint64
	FreeBytes  uint64
}

// DeviceStorage returns the iPhone's AFC-reported storage snapshot.
// Useful for the Dashboard's storage gauge.
func (c *Client) DeviceStorage() (Storage, error) {
	info, err := c.inner.DeviceInfo()
	if err != nil {
		return Storage{}, err
	}
	return Storage{
		Model:      info.Model,
		TotalBytes: info.TotalBytes,
		FreeBytes:  info.FreeBytes,
	}, nil
}

// pickDevice resolves a UDID to an ios.DeviceEntry. With udid=="", pick the
// single USB-connected device; with multiple connected, return an error
// asking the caller to disambiguate.
func pickDevice(udid string) (ios.DeviceEntry, error) {
	list, err := ios.ListDevices()
	if err != nil {
		return ios.DeviceEntry{}, fmt.Errorf("usbmuxd unreachable: %w", err)
	}
	if len(list.DeviceList) == 0 {
		return ios.DeviceEntry{}, errors.New("no devices connected — plug in an iPhone over USB and trust this computer")
	}

	if udid != "" {
		for _, d := range list.DeviceList {
			if d.Properties.SerialNumber == udid {
				return ios.GetDevice(udid)
			}
		}
		return ios.DeviceEntry{}, fmt.Errorf("device with UDID %s not connected", udid)
	}

	// Prefer USB-connected over network-connected.
	usb := list.DeviceList[:0]
	for _, d := range list.DeviceList {
		if d.Properties.ConnectionType == "USB" {
			usb = append(usb, d)
		}
	}
	if len(usb) == 1 {
		return ios.GetDevice(usb[0].Properties.SerialNumber)
	}
	if len(usb) > 1 {
		return ios.DeviceEntry{}, errors.New("multiple USB devices — pass --udid to disambiguate")
	}
	// No USB devices, but network ones exist — use first network device.
	if len(list.DeviceList) == 1 {
		return ios.GetDevice(list.DeviceList[0].Properties.SerialNumber)
	}
	return ios.DeviceEntry{}, errors.New("multiple devices — pass --udid to disambiguate")
}

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

// Entry is one item returned by List — directory or regular file.
//
// Unreadable=true marks entries the device exposed via List() but
// rejected on Stat() — typical of iOS 15+ permission cliffs (e.g.
// /PhotoData/Metadata, app sandboxes without UIFileSharingEnabled).
// The frontend renders these greyed-out so users know the folder
// isn't actually empty.
type Entry struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	IsDir      bool   `json:"is_dir"`
	Unreadable bool   `json:"unreadable,omitempty"`
}

// List returns the immediate children of remoteDir (one level deep, not
// recursive). Hidden entries (basename starting with ".") and the "."
// / ".." synthetic entries are dropped. Result is sorted: directories
// first (alphabetical), then files (alphabetical).
//
// Used by the Browse tab to render an iOS-Files-app-like view of the
// AFC tree (/DCIM, /Books, etc., scoped to com.apple.afc's media jail).
func (c *Client) List(remoteDir string) ([]Entry, error) {
	if remoteDir == "" {
		remoteDir = "/"
	}
	names, err := c.inner.List(remoteDir)
	if err != nil {
		return nil, humanizeAFCError(remoteDir, err)
	}
	out := make([]Entry, 0, len(names))
	for _, n := range names {
		if n == "." || n == ".." || strings.HasPrefix(n, ".") {
			continue
		}
		full := remoteDir
		if !strings.HasSuffix(full, "/") {
			full += "/"
		}
		full += n
		info, err := c.inner.Stat(full)
		if err != nil {
			// Unstatable: usually a permission cliff on iOS 15+ subtrees
			// (/PhotoData/Metadata, app sandboxes without UIFileSharingEnabled,
			// etc.). Keep the row but mark it Unreadable so the GUI can grey
			// it out instead of silently dropping — that way "empty dir"
			// only means actually empty.
			out = append(out, Entry{
				Name:       n,
				Path:       full,
				Unreadable: true,
			})
			continue
		}
		out = append(out, Entry{
			Name:  n,
			Path:  full,
			Size:  info.Size,
			IsDir: info.IsDir(),
		})
	}
	// Dirs first, then alphabetical within each group.
	sortEntries(out)
	return out, nil
}

// humanizeAFCError takes a go-ios `afc error code: N` and turns it
// into something a user can act on. Codes come from libimobiledevice's
// AFC_E_* enum; see https://github.com/libimobiledevice/libimobiledevice
// (src/afc.c). The most common ones the GUI hits are 8 (NOT_FOUND),
// 10 (PERM_DENIED — phone locked or iOS-15+ hardened subtree), 15
// (OP_NOT_SUPPORTED) and 20 (IO_ERROR).
func humanizeAFCError(path string, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "afc error code: 8"):
		return fmt.Errorf("%s: not found (the path doesn't exist on the device)", path)
	case strings.Contains(msg, "afc error code: 9"):
		return fmt.Errorf("%s: that path is a directory (expected a file)", path)
	case strings.Contains(msg, "afc error code: 10"):
		return fmt.Errorf("%s: permission denied. Unlock your iPhone (Face ID / passcode) and try again. If it's already unlocked, this subtree is hardened by iOS 15+ (e.g. /PhotoData/Metadata, app sandboxes without UIFileSharingEnabled) and can't be browsed via AFC", path)
	case strings.Contains(msg, "afc error code: 11"):
		return fmt.Errorf("%s: the AFC service isn't connected. Unplug and replug the iPhone, re-tap Trust if prompted", path)
	case strings.Contains(msg, "afc error code: 13"):
		return fmt.Errorf("%s: too much data (read buffer overflow). Try a narrower path", path)
	case strings.Contains(msg, "afc error code: 15"):
		return fmt.Errorf("%s: that operation isn't supported by AFC on this iOS version", path)
	case strings.Contains(msg, "afc error code: 16"):
		return fmt.Errorf("%s: a file with that name already exists at the destination", path)
	case strings.Contains(msg, "afc error code: 18"):
		return fmt.Errorf("%s: no space left on the device", path)
	case strings.Contains(msg, "afc error code: 20"):
		return fmt.Errorf("%s: I/O error talking to the device. Unplug and replug; try a different USB cable/port if it persists", path)
	}
	return fmt.Errorf("AFC list %s: %w", path, err)
}

// sortEntries orders dirs first then files, alphabetical within each.
func sortEntries(es []Entry) {
	// Tiny n — insertion sort, no need to drag in sort.SliceStable.
	for i := 1; i < len(es); i++ {
		for j := i; j > 0; j-- {
			a, b := es[j-1], es[j]
			less := false
			if a.IsDir != b.IsDir {
				less = b.IsDir
			} else {
				less = strings.ToLower(b.Name) < strings.ToLower(a.Name)
			}
			if !less {
				break
			}
			es[j-1], es[j] = es[j], es[j-1]
		}
	}
}

// Remove deletes a single regular file on the device. Use only after the
// local copy has been verified — there is no recycle bin on iOS.
func (c *Client) Remove(remotePath string) error {
	return c.inner.Remove(remotePath)
}

// RemoveAll recursively deletes a path on the device (file or directory).
// Used by the Browse tab's row-level Delete action when the target is a
// folder. Same data-loss caveat as Remove: no recycle bin on iOS.
func (c *Client) RemoveAll(remotePath string) error {
	return c.inner.RemoveAll(remotePath)
}

// FileInfo is the minimal stat snapshot the rest of DumpSock needs.
// Mirrors go-ios's afc.FileInfo but keeps our public surface stable.
type FileInfo struct {
	Name  string
	Size  int64
	isDir bool
}

func (fi FileInfo) IsDir() bool { return fi.isDir }

// Stat returns FileInfo for a single remote path. Used by the selective
// pull path (Browse-tab "Backup selected") to size entries without a
// full walk.
func (c *Client) Stat(remotePath string) (FileInfo, error) {
	info, err := c.inner.Stat(remotePath)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Name:  info.Name,
		Size:  info.Size,
		isDir: info.IsDir(),
	}, nil
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

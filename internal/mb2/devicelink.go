// Package mb2 — DeviceLink framing layer (Layer 1 of the MobileBackup2 stack).
//
// DeviceLink is the message envelope that wraps every interaction with
// com.apple.mobilebackup2 (and a handful of other Apple lockdown
// services). Each frame on the wire is a 4-byte big-endian length
// prefix followed by a plist whose top-level value is an array. The
// array's first element is the message name (e.g. "DLMessageVersion-
// Exchange", "DLMessageDownloadFiles"); subsequent elements are
// per-message arguments.
//
// We use go-ios's PlistCodec for the length-prefix framing — its wire
// format is byte-identical to what mobilebackup2 expects (4-byte BE
// length + XML plist; iOS parses both XML and binary plist forms).
package mb2

import (
	"errors"
	"fmt"
	"io"

	"github.com/danielpaulus/go-ios/ios"
)

// dlConn wraps a lockdown service connection with DeviceLink framing.
type dlConn struct {
	conn  ios.DeviceConnectionInterface
	codec ios.PlistCodec
}

// newDLConn wraps an already-opened service connection.
func newDLConn(conn ios.DeviceConnectionInterface) *dlConn {
	return &dlConn{conn: conn, codec: ios.NewPlistCodec()}
}

// Close releases the underlying socket. Safe to call after a partial
// failure during the handshake.
func (d *dlConn) Close() error {
	if d == nil || d.conn == nil {
		return nil
	}
	return d.conn.Close()
}

// sendFrame marshals frame (which must be an array — top-level value
// of every DeviceLink message) to a plist and writes it with the
// length prefix.
func (d *dlConn) sendFrame(frame []interface{}) error {
	if len(frame) == 0 {
		return errors.New("sendFrame: empty frame")
	}
	if _, ok := frame[0].(string); !ok {
		return fmt.Errorf("sendFrame: first element must be the message name string, got %T", frame[0])
	}
	encoded, err := d.codec.Encode(frame)
	if err != nil {
		return fmt.Errorf("encode plist: %w", err)
	}
	return d.conn.Send(encoded)
}

// readFrame reads the next length-prefixed plist and returns its
// top-level array. Returns an error if the payload isn't an array.
func (d *dlConn) readFrame() ([]interface{}, error) {
	payload, err := d.codec.Decode(d.conn.Reader())
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("device closed the DeviceLink stream (likely refused our handshake): %w", err)
		}
		return nil, fmt.Errorf("decode plist: %w", err)
	}
	var arr []interface{}
	if _, err := plistUnmarshalArray(payload, &arr); err != nil {
		return nil, fmt.Errorf("DeviceLink frame is not an array: %w", err)
	}
	return arr, nil
}

// frameName pulls the message-name string out of a DeviceLink frame
// (the array's first element). Returns "" if the frame is empty or
// shaped wrong.
func frameName(frame []interface{}) string {
	if len(frame) == 0 {
		return ""
	}
	name, _ := frame[0].(string)
	return name
}

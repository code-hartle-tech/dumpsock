package mb2

import (
	"bytes"

	"howett.net/plist"
)

// plistUnmarshalArray decodes a plist (XML or binary) into a Go
// []interface{}. iOS speaks both formats; the decoder auto-detects via
// the "bplist00" magic header on the binary form. The detected format
// is returned as a string ("binary" or "xml") for tracing — we don't
// branch on it today but it makes future "why did this break"
// debugging cheaper.
func plistUnmarshalArray(data []byte, dst *[]interface{}) (string, error) {
	if err := plist.NewDecoder(bytes.NewReader(data)).Decode(dst); err != nil {
		return "", err
	}
	if len(data) >= 6 && bytes.Equal(data[:6], []byte("bplist")) {
		return "binary", nil
	}
	return "xml", nil
}

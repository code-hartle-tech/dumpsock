//go:build !darwin

// Package macauth — non-darwin stubs. Linux/Windows builds get
// always-unavailable results; the GUI greys out the Touch ID checkbox.
package macauth

import "errors"

var errNotMac = errors.New("biometric vault: macOS only")

func Available() bool                  { return false }
func StorePassword(password string) error { return errNotMac }
func LoadPassword(reason string) (string, error) { return "", errNotMac }
func HasPassword() bool                { return false }
func ClearPassword() error             { return errNotMac }

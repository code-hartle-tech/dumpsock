// Package icloud is the (scaffold) iCloud Photos client. Phase 7 of
// the DumpSock roadmap. Today every entry point returns
// ErrNotImplemented. The intended shape and protocol are documented in
// docs/phase-6-roadmap.md.
//
// Auth landscape (Q1 2026): Apple ID + app-specific password no longer
// works for iCloud Photos. icloudpd switched to SRP-6a (the same
// challenge protocol the Apple website uses) in 2024. A Go port of
// pyicloud_ipd's auth layer (~3.5kLOC) is the first milestone; CloudKit
// Web Services photo queries follow.
//
// Transport: setup.icloud.com for the auth dance, then per-account
// pNN-ckdatabasews.icloud.com for CloudKit query+download. Per-asset
// content URLs come from pNN-content.icloud.com.
//
// Storage layout: identical to dumpsock pull's YYYY-MM-DD/ tree, so
// the two outputs are interchangeable on disk.
package icloud

import "errors"

// ErrNotImplemented marks every scaffold entry point until Phase 7
// lands. Callers can wrap it for nicer error messages.
var ErrNotImplemented = errors.New("icloud: not yet implemented (Phase 7 in development; see docs/phase-6-roadmap.md)")

// Options will mirror backup.Options where it makes sense (OutputRoot,
// Since/Until, Parallel, OnProgress) and add iCloud-specific knobs
// (AppleID, TrustToken path, …) once the auth flow is in place.
type Options struct {
	AppleID    string
	OutputRoot string
	// More to come.
}

// Pull will mirror backup.Run's contract: pulls eligible iCloud Photos
// assets into OutputRoot, returning a Result-shaped summary. Today it
// just returns ErrNotImplemented.
func Pull(opts Options) error {
	_ = opts
	return ErrNotImplemented
}

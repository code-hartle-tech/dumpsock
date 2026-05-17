package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/code-hartle-tech/dumpsock/internal/mb2"
)

// newBackupDeviceCmd builds the `dumpsock backup-device` subcommand.
// Phase 6 scope expansion (2026-05-17): full-device backup via the
// com.apple.mobilebackup2 lockdownd service — the same protocol
// Finder/iTunes uses. Currently a scaffold; the protocol port is the
// largest single workitem in the project. See docs/phase-6-roadmap.md.
func newBackupDeviceCmd(stdout, stderr io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "backup-device",
		Short: "Full-device backup via MobileBackup2 (in development — Phase 6)",
		Long: `Full-device backup.

Produces a Finder/iTunes-equivalent backup tree (Manifest.db,
Manifest.plist, Info.plist, Status.plist, hashed-blob storage)
via the com.apple.mobilebackup2 lockdownd protocol.

Status: in development. go-ios v1.0.213 has no MobileBackup2
package; a pure-Go port of libimobiledevice's idevicebackup2.c
is the next milestone (~3-5kLOC, 2-4 weeks). The local-AFC
photo pull (dumpsock pull) covers 95% of users' actual needs
in the meantime.

Plan: docs/phase-6-roadmap.md
Skeleton: internal/mb2/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(stderr, "backup-device: not yet implemented (Phase 6 in development).")
			fmt.Fprintln(stderr, "See docs/phase-6-roadmap.md for the plan.")
			return mb2.ErrNotImplemented
		},
	}
}

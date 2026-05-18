package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/code-hartle-tech/dumpsock/internal/mb2"
)

// newBackupDeviceCmd builds the `dumpsock backup-device` subcommand.
// Phase 6 scope expansion (2026-05-17): full-device backup via the
// com.apple.mobilebackup2 lockdownd service. Layer 1 (DeviceLink
// framing + service handshake) ships first; the dispatcher loop and
// manifest writers follow. See docs/phase-6-roadmap.md.
func newBackupDeviceCmd(stdout, stderr io.Writer) *cobra.Command {
	var (
		probe bool
		udid  string
	)
	cmd := &cobra.Command{
		Use:   "backup-device",
		Short: "Full-device backup via MobileBackup2 (Phase 6 — Layer 1: --probe works)",
		Long: `Full-device backup.

Produces a Finder/iTunes-equivalent backup tree (Manifest.db,
Manifest.plist, Info.plist, Status.plist, hashed-blob storage)
via the com.apple.mobilebackup2 lockdownd protocol.

Status (2026-05-18): Layer 1 (DeviceLink framing + version
exchange handshake) is live. Run with --probe to confirm the
wire path opens against your device. The full backup flow
(dispatcher loop + manifest writers + file transfer) is the
next milestone.

Plan: docs/phase-6-roadmap.md
Code: internal/mb2/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if probe {
				h, err := mb2.Probe(udid)
				if err != nil {
					return fmt.Errorf("probe failed: %w", err)
				}
				fmt.Fprintf(stdout, "OK. Negotiated MobileBackup2 protocol %d.%d.\n",
					h.NegotiatedMajor, h.NegotiatedMinor)
				if h.ProtocolReady {
					fmt.Fprintln(stdout, "Device replied with DLMessageDeviceReady — the channel is open.")
				}
				return nil
			}
			fmt.Fprintln(stderr, "backup-device: full flow not yet implemented (Phase 6 Layer 2+ in development).")
			fmt.Fprintln(stderr, "Try `dumpsock backup-device --probe` to confirm Layer 1 connects.")
			fmt.Fprintln(stderr, "See docs/phase-6-roadmap.md for the plan.")
			return mb2.ErrNotImplemented
		},
	}
	cmd.Flags().BoolVar(&probe, "probe", false, "Negotiate the MobileBackup2 protocol and exit (Layer 1 connectivity check).")
	cmd.Flags().StringVar(&udid, "udid", "", "Target device UDID (empty = the only attached device).")
	return cmd
}

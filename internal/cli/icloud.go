package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/code-hartle-tech/dumpsock/internal/icloud"
)

// newICloudCmd builds the `dumpsock icloud` subcommand tree. Phase 7
// scope expansion (2026-05-17): read-only iCloud Photos download,
// designed to mirror icloudpd's UX. Currently a scaffold that returns
// ErrNotImplemented — see docs/phase-6-roadmap.md for the auth flow,
// protocol shape, and implementation plan.
func newICloudCmd(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "icloud",
		Short: "Download from iCloud Photos (in development — Phase 7)",
		Long: `iCloud Photos download.

DumpSock's local-AFC path (dumpsock pull) is its first-class flow.
This subcommand brings parity with icloudpd: pull photos+videos
from the cloud, same on-disk YYYY-MM-DD/ layout, no extra database.

Status: in development. The auth/transport flow is documented at
docs/phase-6-roadmap.md and the skeleton lives at
internal/icloud/. Apple switched to SRP-6a auth in 2024; porting
pyicloud_ipd's auth layer is the next milestone.`,
	}
	cmd.AddCommand(newICloudPullCmd(stdout, stderr))
	return cmd
}

func newICloudPullCmd(stdout, stderr io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Download iCloud Photos into a YYYY-MM-DD/ tree",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(stderr, "icloud pull: not yet implemented (Phase 7 in development).")
			fmt.Fprintln(stderr, "See docs/phase-6-roadmap.md for the plan.")
			return icloud.ErrNotImplemented
		},
	}
}

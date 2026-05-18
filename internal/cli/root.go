// Package cli wires up the Cobra command tree.
package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/code-hartle-tech/dumpsock/internal/branding"
)

// NewRootCmd builds the top-level `dumpsock` command. Errors during construction
// are surfaced via the caller; runtime errors come back via Execute().
func NewRootCmd(stdout, stderr io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "dumpsock",
		Short:         branding.Tagline,
		Long:          longDescription(),
		SilenceUsage:  true,
		SilenceErrors: false,
		Version:       fullVersionString(),
	}

	// Custom version template that includes the brand footer.
	root.SetVersionTemplate(versionTemplate())

	// Apple-style minimal default help: surface common commands; advanced
	// flags hidden until --help-advanced or the command-specific --help.
	root.SetHelpTemplate(helpTemplate())

	root.AddCommand(newDevicesCmd(stdout, stderr))
	root.AddCommand(newPullCmd(stdout, stderr))
	root.AddCommand(newVersionCmd(stdout))
	root.AddCommand(newDecryptCmd(stdout, stderr))
	// Phase 6-7 scope expansion (2026-05-17). Both subcommands are
	// scaffolds today; runtime returns ErrNotImplemented.
	root.AddCommand(newICloudCmd(stdout, stderr))
	root.AddCommand(newBackupDeviceCmd(stdout, stderr))

	root.SetOut(stdout)
	root.SetErr(stderr)
	return root
}

func longDescription() string {
	return branding.Tagline + `

DumpSock is a local, offline phone-to-disk media tool. One executable, no
cloud, no account, no telemetry. Plug your iPhone in, run ` + "`dumpsock pull`" + `,
get a date-sorted folder of photos and videos on disk.

` + branding.Footer
}

func fullVersionString() string {
	if branding.Commit != "" {
		return fmt.Sprintf("%s (%s)", branding.Version, branding.Commit)
	}
	return branding.Version
}

func versionTemplate() string {
	return `dumpsock {{.Version}}
` + branding.Footer + `
`
}

func helpTemplate() string {
	return `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
}

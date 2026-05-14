package cli

import (
	"fmt"
	"io"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/code-hartle-tech/dumpsock/internal/branding"
)

func newVersionCmd(stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, build info, and brand line",
		RunE: func(_ *cobra.Command, _ []string) error {
			ver := branding.Version
			if branding.Commit != "" {
				ver = fmt.Sprintf("%s (%s)", ver, branding.Commit)
			}
			fmt.Fprintf(stdout, "dumpsock %s\n", ver)
			fmt.Fprintf(stdout, "%s/%s · go%s\n",
				runtime.GOOS, runtime.GOARCH, runtime.Version()[2:])
			fmt.Fprintf(stdout, "%s\n", branding.Footer)
			return nil
		},
	}
}

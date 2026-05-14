package cli

import (
	"errors"
	"io"

	"github.com/spf13/cobra"
)

// pullOpts captures all flags. Day-1 flags are visible; advanced flags are
// marked Hidden so the default --help stays Apple-tier minimal.
type pullOpts struct {
	output      string
	since       string
	until       string
	deleteAfter bool
	confirmDel  bool
	watchSecs   int
	dryRun      bool

	// advanced
	parallel    int
	untilFound  int
	noMtime     bool
	noLivePair  bool
	noNotify    bool
	hashMode    string
	remoteRoot  string
	jsonStream  bool
}

func newPullCmd(stdout, stderr io.Writer) *cobra.Command {
	o := pullOpts{
		parallel:   4,
		hashMode:   "size",
		remoteRoot: "DCIM",
	}

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Copy photos and videos from the connected iPhone to disk",
		Long: `Walk the iPhone's DCIM tree over USB, copy each file to a date-sorted
folder on disk, skip what's already present (matched by filename and byte
size), and never silently overwrite.

The default output is ~/DumpSock/<device-name>/ and the on-disk layout is
identical to icloudpd's: YYYY-MM-DD/ subfolders by capture date, with a
0000:00:00 00:00:00/ bucket for files whose date can't be read.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if o.deleteAfter && !o.confirmDel {
				return errors.New("--delete-after requires --confirm-delete on the same invocation")
			}
			return errors.New("pull: backup engine not yet wired (feature/cli-skeleton — engine lands in feature/pull-engine)")
		},
	}

	// Day-1 flags
	cmd.Flags().StringVarP(&o.output, "output", "o", "", "destination root (default: ~/DumpSock/<device-name>)")
	cmd.Flags().StringVar(&o.since, "since", "", "only pull files captured on/after YYYY-MM-DD")
	cmd.Flags().StringVar(&o.until, "until", "", "only pull files captured on/before YYYY-MM-DD")
	cmd.Flags().BoolVar(&o.deleteAfter, "delete-after", false, "remove files from the device after verified copy (requires --confirm-delete)")
	cmd.Flags().BoolVar(&o.confirmDel, "confirm-delete", false, "explicit confirmation gate for --delete-after")
	cmd.Flags().IntVar(&o.watchSecs, "watch", 0, "stay running and rescan every N seconds (0 = one-shot)")
	cmd.Flags().BoolVar(&o.dryRun, "dry-run", false, "plan only — list what would be pulled, transfer nothing")

	// Advanced flags — hidden from default --help
	cmd.Flags().IntVar(&o.parallel, "parallel", 4, "concurrent file pulls")
	cmd.Flags().IntVar(&o.untilFound, "until-found", 0, "stop after N consecutive name+size matches against existing dest")
	cmd.Flags().BoolVar(&o.noMtime, "no-mtime", false, "don't set file mtime to capture date")
	cmd.Flags().BoolVar(&o.noLivePair, "no-live-pair", false, "don't group Live Photo HEIC+MOV pairs into the same date folder")
	cmd.Flags().BoolVar(&o.noNotify, "no-notify", false, "don't fire a desktop notification on completion")
	cmd.Flags().StringVar(&o.hashMode, "hash", "size", "dedup mode: size | sha256")
	cmd.Flags().StringVar(&o.remoteRoot, "remote-root", "DCIM", "path on the device to walk (advanced)")
	cmd.Flags().BoolVar(&o.jsonStream, "json", false, "machine-readable progress stream on stdout")

	for _, f := range []string{
		"parallel", "until-found", "no-mtime", "no-live-pair",
		"no-notify", "hash", "remote-root", "json",
	} {
		_ = cmd.Flags().MarkHidden(f)
	}

	return cmd
}

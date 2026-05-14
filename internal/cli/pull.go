package cli

import (
	"errors"
	"fmt"
	"io"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/code-hartle-tech/dumpsock/internal/backup"
)

type pullOpts struct {
	udid        string
	output      string
	since       string
	until       string
	deleteAfter bool
	confirmDel  bool
	watchSecs   int
	dryRun      bool

	// advanced
	parallel   int
	untilFound int
	noMtime    bool
	noLivePair bool
	noNotify   bool
	hashMode   string
	remoteRoot string
	jsonStream bool
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			if o.deleteAfter && !o.confirmDel {
				return errors.New("--delete-after requires --confirm-delete on the same invocation")
			}
			if o.hashMode != "size" && o.hashMode != "sha256" {
				return fmt.Errorf("--hash must be 'size' or 'sha256' (got %q)", o.hashMode)
			}
			if o.hashMode == "sha256" {
				fmt.Fprintln(stderr, "warning: --hash sha256 not yet implemented; using size+name")
			}

			since, err := parseDateFlag(o.since, "--since")
			if err != nil {
				return err
			}
			until, err := parseDateFlag(o.until, "--until")
			if err != nil {
				return err
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			opts := backup.Options{
				UDID:        o.udid,
				OutputRoot:  o.output,
				RemoteRoot:  o.remoteRoot,
				Since:       since,
				Until:       until,
				DeleteAfter: o.deleteAfter,
				DryRun:      o.dryRun,
				Parallel:    o.parallel,
				UntilFound:  o.untilFound,
				SetMtime:    !o.noMtime,
				Notify:      !o.noNotify,
				JSONStream:  o.jsonStream,
				Out:         stdout,
				Err:         stderr,
			}

			runOnce := func() error {
				res, err := backup.Run(ctx, opts)
				if err != nil {
					return err
				}
				fmt.Fprintf(stdout,
					"done: total=%d pre_skipped=%d pulled=%d post_skipped=%d suffixed=%d nodate=%d filtered=%d errors=%d (%s)\n",
					res.Total, res.PreSkipped, res.Pulled, res.PostSkipped,
					res.Suffixed, res.NoDate, res.Filtered, res.Errors,
					res.Elapsed.Round(time.Second))
				return nil
			}

			if o.watchSecs <= 0 {
				return runOnce()
			}
			fmt.Fprintf(stdout, "watch mode: rescan every %ds (Ctrl-C to stop)\n", o.watchSecs)
			for {
				if err := runOnce(); err != nil {
					fmt.Fprintf(stderr, "watch: %v\n", err)
				}
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(time.Duration(o.watchSecs) * time.Second):
				}
			}
		},
	}

	cmd.Flags().StringVar(&o.udid, "udid", "", "device UDID (only needed if multiple iPhones are connected)")
	cmd.Flags().StringVarP(&o.output, "output", "o", "", "destination root (default: ~/DumpSock/<device-name>)")
	cmd.Flags().StringVar(&o.since, "since", "", "only pull files captured on/after YYYY-MM-DD")
	cmd.Flags().StringVar(&o.until, "until", "", "only pull files captured on/before YYYY-MM-DD")
	cmd.Flags().BoolVar(&o.deleteAfter, "delete-after", false, "remove files from device after verified copy (requires --confirm-delete)")
	cmd.Flags().BoolVar(&o.confirmDel, "confirm-delete", false, "explicit confirmation gate for --delete-after")
	cmd.Flags().IntVar(&o.watchSecs, "watch", 0, "stay running and rescan every N seconds (0 = one-shot)")
	cmd.Flags().BoolVar(&o.dryRun, "dry-run", false, "plan only — list what would be pulled, transfer nothing")

	cmd.Flags().IntVar(&o.parallel, "parallel", 4, "concurrent post-pull workers (EXIF + move)")
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

func parseDateFlag(s, name string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: expected YYYY-MM-DD, got %q", name, s)
	}
	return t, nil
}

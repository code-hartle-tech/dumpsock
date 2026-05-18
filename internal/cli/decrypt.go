package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/code-hartle-tech/dumpsock/internal/backup"
)

// newDecryptCmd builds `dumpsock decrypt <file.zip.aes> [-o output.zip]`.
// Reads DSAES2 (current) or DSAES1 (legacy) format.
//
// Password sources, in order:
//  1. --password / -p flag
//  2. DUMPSOCK_PASSWORD env var (so shell scripts don't have to TTY-prompt)
//  3. Interactive prompt (terminal — hidden input)
func newDecryptCmd(stdout, stderr io.Writer) *cobra.Command {
	var (
		password string
		outPath  string
	)
	cmd := &cobra.Command{
		Use:   "decrypt <file.zip.aes>",
		Short: "Decrypt a DumpSock-encrypted archive back to a plain .zip",
		Long: `Decrypt a DumpSock-encrypted archive.

Accepts both the current DSAES2 (chunked, streaming) format and the
legacy single-shot DSAES1 format. Output is a vanilla .zip you can
drag into Finder, extract with 7-Zip/Keka/unzip, or feed back into
DumpSock as an external source.

Password sources are tried in this order: --password flag,
$DUMPSOCK_PASSWORD env var, then interactive prompt.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := args[0]
			if _, err := os.Stat(src); err != nil {
				return fmt.Errorf("source not readable: %w", err)
			}
			pw := password
			if pw == "" {
				pw = os.Getenv("DUMPSOCK_PASSWORD")
			}
			if pw == "" {
				p, err := promptPassword(stderr)
				if err != nil {
					return err
				}
				pw = p
			}
			if outPath == "" {
				outPath = strings.TrimSuffix(src, ".aes")
				if outPath == src {
					outPath = src + ".decrypted"
				}
			}
			if _, err := os.Stat(outPath); err == nil {
				return fmt.Errorf("output already exists: %s (pass -o to choose a different name)", outPath)
			}
			fmt.Fprintf(stderr, "decrypting %s → %s …\n", filepath.Base(src), filepath.Base(outPath))
			final, err := backup.DecryptFile(src, outPath, pw)
			if err != nil {
				return err
			}
			fi, _ := os.Stat(final)
			size := int64(0)
			if fi != nil {
				size = fi.Size()
			}
			fmt.Fprintf(stdout, "Wrote %s (%d bytes).\n", final, size)
			return nil
		},
	}
	cmd.Flags().StringVarP(&password, "password", "p", "", "decryption password (otherwise read from DUMPSOCK_PASSWORD env or prompted)")
	cmd.Flags().StringVarP(&outPath, "output", "o", "", "output path (default: drop .aes suffix from input)")
	return cmd
}

// promptPassword reads a password without echoing it. Uses
// golang.org/x/term.ReadPassword when stderr is a terminal; falls back
// to plain stdin read otherwise.
func promptPassword(stderr io.Writer) (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(stderr, "Password: ")
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(stderr)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}
	// Non-TTY (piped). Read one line.
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

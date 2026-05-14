// Command dumpsock is the DumpSock CLI entry point.
//
// DumpSock — by HARTLE.TECH · contact@hartle.tech
package main

import (
	"fmt"
	"os"

	"github.com/code-hartle-tech/dumpsock/internal/cli"
)

func main() {
	root := cli.NewRootCmd(os.Stdout, os.Stderr)
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "dumpsock: %s\n", err)
		os.Exit(1)
	}
}

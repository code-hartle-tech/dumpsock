// Package branding holds the static brand strings injected throughout the CLI.
// Per CLAUDE.md voice rules: HARTLE.TECH only, contact@hartle.tech only, no
// operator name, no exclamation marks, confident-deadpan tone.
package branding

// Tagline appears under the root command's help. Single-line, no period.
const Tagline = "Plug your phone in. Wring it dry."

// Footer appears at the bottom of help output and --version output.
const Footer = "A HARTLE.TECH tool · contact@hartle.tech"

// Version is set via -ldflags at build time. Defaults to "dev" for local builds.
var Version = "dev"

// Commit is the short git SHA injected at build time. "" when unset.
var Commit = ""

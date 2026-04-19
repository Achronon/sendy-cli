// sendy — CLI for sendy.md. Paste terminal output, get a markdown link.
package main

import (
	"fmt"
	"io"
	"os"
)

// version is overridden at build time via `-ldflags "-X main.version=<tag>"`.
// See .github/workflows/release-cli.yml.
var version = "0.1.1-dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	// Backwards compat: `sendy FILE`, `sendy -`, `command | sendy`
	// still creates a paste — same as the old bash script. If the first
	// positional arg is not a known subcommand, fall through to create.
	if len(args) == 0 {
		return cmdCreate(args, stdin, stdout, stderr)
	}

	switch args[0] {
	case "-h", "--help", "help":
		printUsage(stdout)
		return 0
	case "-v", "--version", "version":
		fmt.Fprintln(stdout, version)
		return 0
	case "create":
		return cmdCreate(args[1:], stdin, stdout, stderr)
	case "list", "ls":
		return cmdList(args[1:], stdout, stderr)
	case "view", "cat":
		return cmdView(args[1:], stdout, stderr)
	case "raw":
		return cmdRaw(args[1:], stdout, stderr)
	case "claim":
		return cmdClaim(args[1:], stdout, stderr)
	case "whoami":
		return cmdWhoami(stdout, stderr)
	case "login":
		return cmdLogin(args[1:], stdout, stderr)
	case "logout":
		return cmdLogout(args[1:], stdout, stderr)
	case "completions":
		return cmdCompletions(args[1:], stdout, stderr)
	}

	// Fallback: treat unknown first arg as a file path for create, matching
	// the old bash CLI's ergonomics (`sendy path/to/file`).
	return cmdCreate(args, stdin, stdout, stderr)
}

const usage = `sendy — paste to sendy.md, get a markdown link

Usage:
  sendy [FILE|-]              Create paste from file, stdin, or ` + "`-`" + `
  sendy create [FILE|-]       Explicit create
  sendy list [--limit N] [--search Q]   List your pastes (Q filters by content)
  sendy view <slug>           Print a paste's content
  sendy raw <slug>            Print raw text of a paste
  sendy login                 Open a browser to sign in (stores token in OS keyring)
  sendy logout                Remove the stored session token
  sendy claim [--user-key K]  Claim anonymous pastes under the signed-in account
  sendy whoami                Show configured identity
  sendy completions SHELL     Print completion script for bash, zsh, or fish
  sendy help                  Show this help
  sendy version               Print version

Flags (create):
  --password PASS             Password-protect the paste
  --user-key KEY              Override the user_key for this paste

Flags (view):
  --password PASS             Unlock a password-protected paste

Global env:
  SENDY_URL            API base URL (default: https://sendy.md)
  SENDY_USER_KEY       Identifier for anonymous history scoping (before login)
  SENDY_SESSION_TOKEN  Overrides the keyring token (useful for CI / scripts)

Examples:
  echo hi | sendy
  sendy login
  sendy list --limit 10
  sendy view abc123
  sendy claim --user-key myalice
  sendy completions zsh > ~/.zsh/completions/_sendy
`

func printUsage(w io.Writer) {
	fmt.Fprint(w, usage)
}

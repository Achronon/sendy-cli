package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// cmdClaim re-assigns pastes created with a given user_key to the
// authenticated session user. Defaults to SENDY_USER_KEY when no
// explicit --user-key flag is passed; fails loudly if neither is set.
func cmdClaim(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("claim", flag.ContinueOnError)
	fs.SetOutput(stderr)
	userKey := fs.String("user-key", "", "Claim pastes associated with this user_key (defaults to SENDY_USER_KEY)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *userKey == "" {
		*userKey = os.Getenv("SENDY_USER_KEY")
	}
	if *userKey == "" {
		fmt.Fprintln(stderr, "error: pass --user-key KEY or set SENDY_USER_KEY")
		return 1
	}

	client, err := newClient()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	count, err := client.Claim(*userKey)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	fmt.Fprintf(stdout, "Claimed %d paste%s for user_key %s\n",
		count, pluralS(count), *userKey)
	return 0
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

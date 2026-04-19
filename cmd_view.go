package main

import (
	"flag"
	"fmt"
	"io"
)

func cmdView(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("view", flag.ContinueOnError)
	fs.SetOutput(stderr)
	password := fs.String("password", "", "Unlock a password-protected paste")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(stderr, "error: slug required")
		return 1
	}

	client, err := newClient()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	resp, err := client.Fetch(rest[0], *password)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.Code == "PASSWORD_REQUIRED" {
			fmt.Fprintln(stderr, "error: paste is password-protected — pass --password PASS")
			return 1
		}
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	fmt.Fprintln(stdout, resp.Content)
	return 0
}

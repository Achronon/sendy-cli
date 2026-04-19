package main

import (
	"fmt"
	"io"
	"os"
)

func cmdWhoami(stdout, stderr io.Writer) int {
	fmt.Fprintf(stdout, "api:       %s\n", apiBaseURL())

	client, err := newClient()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	if client.isAuthenticated() {
		user, err := client.GetSession()
		if err != nil {
			fmt.Fprintln(stdout, "auth:      token present but session check failed —", err)
			return 0
		}
		if user == nil {
			fmt.Fprintln(stdout, "auth:      token present but server rejected it — run `sendy logout` then `sendy login`")
			return 0
		}
		fmt.Fprintf(stdout, "auth:      signed in as %s\n", user.Email)
	} else {
		fmt.Fprintln(stdout, "auth:      not signed in (run `sendy login`)")
	}

	if k := os.Getenv("SENDY_USER_KEY"); k != "" {
		fmt.Fprintf(stdout, "user_key:  %s (from env)\n", k)
	} else if !client.isAuthenticated() {
		fmt.Fprintln(stdout, "user_key:  (unset — list will fail)")
	}
	return 0
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

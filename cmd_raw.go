package main

import (
	"fmt"
	"io"
)

func cmdRaw(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "error: slug required")
		return 1
	}

	client, err := newClient()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	body, err := client.FetchRaw(args[0])
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	fmt.Fprint(stdout, body)
	return 0
}

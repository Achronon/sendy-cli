package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

func cmdList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	limit := fs.Int("limit", 20, "Max pastes to return")
	offset := fs.Int("offset", 0, "Pagination offset")
	userKey := fs.String("user-key", "", "Override SENDY_USER_KEY")
	search := fs.String("search", "", "Case-insensitive substring filter over paste content")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	client, err := newClient()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if *userKey != "" {
		client.userKey = *userKey
	}

	// When searching, ask the server for full content (not just the
	// 100-char preview) so the substring match is useful. Server-side
	// search is a future API addition; for now we filter locally.
	needFull := *search != ""
	resp, err := client.List(*limit, *offset, needFull)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	pastes := resp.Pastes
	if *search != "" {
		pastes = filterPastes(pastes, *search)
	}

	if len(pastes) == 0 {
		if *search != "" {
			fmt.Fprintf(stdout, "no pastes match %q\n", *search)
		} else {
			fmt.Fprintln(stdout, "no pastes yet")
		}
		return 0
	}

	for _, p := range pastes {
		lock := " "
		if p.Protected {
			lock = "🔒"
		}
		preview := collapseWhitespace(p.Preview)
		if len(preview) > 60 {
			preview = preview[:60] + "…"
		}
		fmt.Fprintf(stdout, "%s  %s  %s %s\n",
			p.CreatedAt.Format("2006-01-02 15:04"),
			p.Slug,
			lock,
			preview,
		)
	}
	if *search != "" {
		fmt.Fprintf(stdout, "\n%d match%s (from %d fetched)\n",
			len(pastes), pluralES(len(pastes)), len(resp.Pastes))
	} else if resp.Total > len(pastes) {
		fmt.Fprintf(stdout, "\n%d of %d total\n", len(pastes), resp.Total)
	}
	return 0
}

func filterPastes(pastes []ListItem, query string) []ListItem {
	q := strings.ToLower(query)
	out := pastes[:0:0]
	for _, p := range pastes {
		if strings.Contains(strings.ToLower(p.Preview), q) {
			out = append(out, p)
		}
	}
	return out
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func pluralES(n int) string {
	if n == 1 {
		return ""
	}
	return "es"
}

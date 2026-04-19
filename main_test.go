package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--help"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d (stderr=%q)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("help output missing Usage header: %q", stdout.String())
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if strings.TrimSpace(stdout.String()) == "" {
		t.Fatal("version output empty")
	}
}

func TestCreateNoInputFails(t *testing.T) {
	// Empty stdin, no file arg → should error out with a helpful message.
	var stdout, stderr bytes.Buffer
	code := cmdCreate([]string{}, strings.NewReader(""), &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit, got 0 (stdout=%q, stderr=%q)",
			stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "no content") {
		t.Fatalf("expected 'no content' in stderr, got %q", stderr.String())
	}
}

func TestCollapseWhitespace(t *testing.T) {
	cases := map[string]string{
		"foo bar":         "foo bar",
		"foo  bar   baz":  "foo bar baz",
		"\tfoo\nbar\r\n":  "foo bar",
		"   ":             "",
	}
	for in, want := range cases {
		if got := collapseWhitespace(in); got != want {
			t.Errorf("collapseWhitespace(%q) = %q, want %q", in, got, want)
		}
	}
}

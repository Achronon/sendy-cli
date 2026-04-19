package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

func cmdCreate(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	password := fs.String("password", "", "Password-protect the paste")
	userKey := fs.String("user-key", "", "Override SENDY_USER_KEY for this paste")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()

	content, err := readContent(rest, stdin)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if content == "" {
		fmt.Fprintln(stderr, "error: no content — pipe something or pass a file")
		printUsage(stderr)
		return 1
	}

	client, err := newClient()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if *userKey != "" {
		client.userKey = *userKey
	}

	resp, err := client.Create(CreateReq{
		Content:  content,
		Password: *password,
		UserKey:  client.userKey,
	})
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	if copied := copyToClipboard(resp.URL); copied {
		fmt.Fprintf(stdout, "%s (copied to clipboard)\n", resp.URL)
	} else {
		fmt.Fprintln(stdout, resp.URL)
	}
	return 0
}

// readContent resolves input from either a file path, stdin (when piped),
// or literal `-` sentinel. Order of precedence matches the old bash CLI.
func readContent(rest []string, stdin io.Reader) (string, error) {
	if len(rest) > 0 && rest[0] != "-" {
		buf, err := os.ReadFile(rest[0])
		if err != nil {
			return "", fmt.Errorf("file not found: %s", rest[0])
		}
		return string(buf), nil
	}

	// stdin path (piped, redirected, or explicit `-`)
	if !isTTY(stdin) || (len(rest) > 0 && rest[0] == "-") {
		buf, err := io.ReadAll(stdin)
		if err != nil {
			return "", err
		}
		return string(buf), nil
	}
	return "", nil
}

// isTTY returns true when the reader is a terminal (no piped input).
// Uses os.File stat; non-file readers are treated as non-TTY so tests
// can pass arbitrary io.Readers.
func isTTY(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// copyToClipboard writes text to the system clipboard, returning true on
// success. macOS uses pbcopy; Linux tries wl-copy then xclip. Silent
// no-op on other platforms.
func copyToClipboard(text string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	}
	if cmd == nil {
		return false
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		return false
	}
	if err := cmd.Start(); err != nil {
		return false
	}
	_, _ = in.Write([]byte(text))
	_ = in.Close()
	return cmd.Wait() == nil
}

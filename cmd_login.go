package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// cmdLogin runs the browser-based sign-in flow:
//
//  1. Bind a localhost listener on an ephemeral port.
//  2. Generate a random `state` nonce + PKCE verifier/challenge pair.
//  3. Open `<api>/auth/native?state=<nonce>&port=<port>&code_challenge=<c>
//     &code_challenge_method=S256` in the system browser.
//  4. User signs in via Better Auth. The server-side page encrypts the
//     session token and the challenge into a short-lived `code` and
//     redirects to `http://127.0.0.1:<port>/callback?code=<code>&state=<nonce>`.
//  5. Our listener captures the request, verifies state, POSTs the code
//     + `code_verifier` to `/api/auth/native-exchange`. Server checks
//     that SHA-256(verifier) == stored challenge before handing over the
//     session token.
//  6. Token goes in the OS keyring; a success page is shown in-browser.
//
// The PKCE leg closes the attack where a malicious localhost listener
// steals the code — without the verifier, a stolen code won't exchange.
func cmdLogin(args []string, stdout, stderr io.Writer) int {
	apiBase := apiBaseURL()

	state, err := randomHex(16)
	if err != nil {
		fmt.Fprintln(stderr, "error: could not generate state:", err)
		return 1
	}

	verifier, challenge, err := newPKCEPair()
	if err != nil {
		fmt.Fprintln(stderr, "error: could not generate PKCE pair:", err)
		return 1
	}

	// 0/0 = kernel-assigned ephemeral port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(stderr, "error: could not bind local callback port:", err)
		return 1
	}
	port := listener.Addr().(*net.TCPAddr).Port

	type callbackResult struct {
		code string
		err  error
	}
	done := make(chan callbackResult, 1)
	var serveOnce sync.Once

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		code := q.Get("code")
		gotState := q.Get("state")
		serveOnce.Do(func() {
			if code == "" {
				done <- callbackResult{err: fmt.Errorf("missing code in callback")}
				renderCallbackPage(w, false, "missing code")
				return
			}
			if gotState != state {
				done <- callbackResult{err: fmt.Errorf("state mismatch: possible CSRF, aborting")}
				renderCallbackPage(w, false, "state mismatch")
				return
			}
			done <- callbackResult{code: code}
			renderCallbackPage(w, true, "")
		})
	})

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	authURL := fmt.Sprintf(
		"%s/auth/native?state=%s&port=%d&code_challenge=%s&code_challenge_method=S256",
		strings.TrimRight(apiBase, "/"),
		url.QueryEscape(state),
		port,
		url.QueryEscape(challenge),
	)

	fmt.Fprintln(stdout, "Opening browser to", authURL)
	fmt.Fprintln(stdout, "Waiting for sign-in to complete…")

	if err := openBrowser(authURL); err != nil {
		fmt.Fprintln(stderr, "could not auto-open browser — open the URL above manually.")
	}

	select {
	case res := <-done:
		if res.err != nil {
			fmt.Fprintln(stderr, "error:", res.err)
			return 1
		}
		return finishLogin(apiBase, res.code, verifier, stdout, stderr)
	case <-time.After(5 * time.Minute):
		fmt.Fprintln(stderr, "error: login timed out after 5 minutes")
		return 1
	}
}

func finishLogin(apiBase, code, verifier string, stdout, stderr io.Writer) int {
	body, _ := json.Marshal(map[string]string{
		"code":          code,
		"code_verifier": verifier,
	})
	req, err := http.NewRequest("POST", strings.TrimRight(apiBase, "/")+"/api/auth/native-exchange", strings.NewReader(string(body)))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		fmt.Fprintln(stderr, "error: token exchange failed:", err)
		return 1
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Fprintf(stderr, "error: exchange returned HTTP %d: %s\n", resp.StatusCode, string(raw))
		return 1
	}

	var exchange struct {
		SessionToken string `json:"session_token"`
		User         struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"user"`
	}
	if err := json.Unmarshal(raw, &exchange); err != nil {
		fmt.Fprintln(stderr, "error: could not parse exchange response:", err)
		return 1
	}
	if exchange.SessionToken == "" {
		fmt.Fprintln(stderr, "error: exchange succeeded but returned no session_token")
		return 1
	}

	if err := saveTokenToKeyring(exchange.SessionToken); err != nil {
		fmt.Fprintln(stderr, "error: could not store token in keyring:", err)
		return 1
	}

	who := exchange.User.Email
	if who == "" {
		who = "user " + exchange.User.ID
	}
	fmt.Fprintf(stdout, "Signed in as %s\n", who)
	return 0
}

func cmdLogout(args []string, stdout, stderr io.Writer) int {
	if err := clearTokenFromKeyring(); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	fmt.Fprintln(stdout, "Signed out.")
	return 0
}

func apiBaseURL() string {
	if u := strings.TrimSpace(os.Getenv("SENDY_URL")); u != "" {
		return u
	}
	return "https://sendy.md"
}

func randomHex(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// newPKCEPair returns a (verifier, challenge) pair per RFC 7636 §4.2.
// The verifier is 32 random bytes encoded as base64url-without-padding
// (43 chars, well within the 43-128 range). The challenge is
// BASE64URL(SHA256(verifier)). Only S256 is supported — plaintext
// challenges defeat PKCE.
func newPKCEPair() (verifier, challenge string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func openBrowser(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "linux":
		cmd = exec.Command("xdg-open", target)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", target)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

func renderCallbackPage(w http.ResponseWriter, ok bool, reason string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if ok {
		_, _ = w.Write([]byte(`<!doctype html><meta charset=utf-8><title>sendy.md</title><style>body{background:#0a0a0a;color:#e5e5e5;font:16px/1.5 ui-monospace,monospace;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}main{text-align:center}h1{color:#fff}h1 span{color:#10b981}p{color:#a3a3a3}</style><main><h1>sendy<span>.md</span></h1><p>Signed in — you can close this tab and return to the terminal.</p></main>`))
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	msg := htmlEscape(reason)
	_, _ = w.Write([]byte(fmt.Sprintf(`<!doctype html><meta charset=utf-8><title>sendy.md — sign-in failed</title><style>body{background:#0a0a0a;color:#e5e5e5;font:16px/1.5 ui-monospace,monospace;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}main{text-align:center}h1{color:#fff}h1 span{color:#10b981}p{color:#f87171}</style><main><h1>sendy<span>.md</span></h1><p>Sign-in failed: %s</p><p>Return to the terminal and try again.</p></main>`, msg)))
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

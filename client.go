package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Client talks to the sendy.md JSON API. Paths use the unversioned /api
// prefix for now; the /api/v1 tree added in the OpenAPI PR re-exports
// the same handlers, so behavior is identical. Switch the constants
// here once /v1 diverges from /api — the spec at /api/openapi.json is
// the source of truth.
type Client struct {
	baseURL      *url.URL
	http         *http.Client
	userKey      string
	sessionToken string
}

func newClient() (*Client, error) {
	raw := os.Getenv("SENDY_URL")
	if raw == "" {
		raw = "https://sendy.md"
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid SENDY_URL: %w", err)
	}
	return &Client{
		baseURL:      u,
		http:         &http.Client{Timeout: 30 * time.Second},
		userKey:      os.Getenv("SENDY_USER_KEY"),
		sessionToken: tokenFromKeyring(),
	}, nil
}

func (c *Client) isAuthenticated() bool { return c.sessionToken != "" }

type APIError struct {
	Status  int
	Code    string `json:"code"`
	Message string `json:"error"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s (HTTP %d)", e.Code, e.Message, e.Status)
	}
	return fmt.Sprintf("%s (HTTP %d)", e.Message, e.Status)
}

type CreateReq struct {
	Content  string `json:"content"`
	Password string `json:"password,omitempty"`
	UserKey  string `json:"user_key,omitempty"`
}

type CreateResp struct {
	Slug      string `json:"slug"`
	URL       string `json:"url"`
	Protected bool   `json:"protected"`
}

func (c *Client) Create(req CreateReq) (*CreateResp, error) {
	// Authenticated clients let the server attach the paste to the user
	// account; sending a user_key too would be redundant.
	if !c.isAuthenticated() && req.UserKey == "" {
		req.UserKey = c.userKey
	}
	var out CreateResp
	if err := c.do("POST", "/api/pastes", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Claim migrates pastes that were created with a given user_key to the
// authenticated session user's account. Requires a session token.
func (c *Client) Claim(userKey string) (int, error) {
	if !c.isAuthenticated() {
		return 0, fmt.Errorf("authentication required (run `sendy login`)")
	}
	var out struct {
		Claimed int `json:"claimed"`
	}
	body := map[string]string{"user_key": userKey}
	if err := c.do("POST", "/api/pastes/claim", nil, body, &out); err != nil {
		return 0, err
	}
	return out.Claimed, nil
}

// GetSession hits Better Auth to validate the stored token and return
// the user record. Used by `sendy whoami`. Returns (nil, nil) when the
// CLI is not authenticated.
type SessionUser struct {
	ID    string
	Email string
	Name  string
}

func (c *Client) GetSession() (*SessionUser, error) {
	if !c.isAuthenticated() {
		return nil, nil
	}
	var out struct {
		User struct {
			ID    string `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"user"`
	}
	if err := c.do("GET", "/api/auth/get-session", nil, nil, &out); err != nil {
		return nil, err
	}
	if out.User.ID == "" {
		return nil, nil
	}
	return &SessionUser{ID: out.User.ID, Email: out.User.Email, Name: out.User.Name}, nil
}

type ListItem struct {
	Slug      string    `json:"slug"`
	Preview   string    `json:"preview"`
	CreatedAt time.Time `json:"created_at"`
	Protected bool      `json:"protected"`
}

type ListResp struct {
	Pastes []ListItem `json:"pastes"`
	Total  int        `json:"total"`
}

func (c *Client) List(limit, offset int, full bool) (*ListResp, error) {
	// An authenticated session is listed by the server against the user
	// account; SENDY_USER_KEY is the fallback for anonymous CLIs.
	if !c.isAuthenticated() && c.userKey == "" {
		return nil, fmt.Errorf("not authenticated — run `sendy login` or set SENDY_USER_KEY")
	}
	q := url.Values{
		"limit":  {strconv.Itoa(limit)},
		"offset": {strconv.Itoa(offset)},
	}
	if !c.isAuthenticated() {
		q.Set("user_key", c.userKey)
	}
	if full {
		q.Set("full", "true")
	}
	var out ListResp
	if err := c.do("GET", "/api/pastes?"+q.Encode(), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type PasteResp struct {
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Protected bool      `json:"protected"`
}

func (c *Client) Fetch(slug, password string) (*PasteResp, error) {
	headers := map[string]string{}
	if password != "" {
		headers["X-Paste-Password"] = password
	}
	var out PasteResp
	if err := c.do("GET", "/api/pastes/"+url.PathEscape(slug), headers, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) FetchRaw(slug string) (string, error) {
	// Hits the rendered site route, not the JSON API — that's the
	// supported contract for "give me just the text".
	req, err := http.NewRequest("GET", c.baseURL.String()+"/"+url.PathEscape(slug)+"/raw", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("fetch raw failed: HTTP %d", resp.StatusCode)
	}
	return string(body), nil
}

// do is the shared request helper. `body` is JSON-encoded when non-nil;
// `out` is JSON-decoded when non-nil. Non-2xx responses are parsed into
// an APIError when the body shape matches { error, code }.
func (c *Client) do(method, path string, headers map[string]string, body, out any) error {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, c.baseURL.String()+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.sessionToken != "" {
		// Matches the cookie name Better Auth sets in production (the
		// Secure prefix is also what the macOS app uses).
		req.Header.Set("Cookie", "__Secure-neon-auth.session_token="+c.sessionToken)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		apiErr := &APIError{Status: resp.StatusCode}
		_ = json.Unmarshal(raw, apiErr)
		if apiErr.Message == "" {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return apiErr
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

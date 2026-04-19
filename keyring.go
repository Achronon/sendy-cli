package main

import (
	"errors"
	"os"

	"github.com/zalando/go-keyring"
)

const (
	keyringService     = "sendy-cli"
	keyringTokenKey    = "session_token"
	keyringUserKeyKey  = "user_key"
)

// tokenFromKeyring returns the stored session token. A non-empty
// SENDY_SESSION_TOKEN env var overrides the keyring — useful for CI,
// scripts, and testing.
func tokenFromKeyring() string {
	if env := os.Getenv("SENDY_SESSION_TOKEN"); env != "" {
		return env
	}
	token, err := keyring.Get(keyringService, keyringTokenKey)
	if err != nil {
		return ""
	}
	return token
}

func saveTokenToKeyring(token string) error {
	return keyring.Set(keyringService, keyringTokenKey, token)
}

func clearTokenFromKeyring() error {
	err := keyring.Delete(keyringService, keyringTokenKey)
	// "secret not found" is not an error from the user's perspective —
	// logout is idempotent.
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return nil
}

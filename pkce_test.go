package main

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestNewPKCEPair(t *testing.T) {
	verifier, challenge, err := newPKCEPair()
	if err != nil {
		t.Fatalf("newPKCEPair: %v", err)
	}
	// RFC 7636: verifier must be 43-128 chars, URL-safe.
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Errorf("verifier length %d out of range [43,128]", len(verifier))
	}
	if _, err := base64.RawURLEncoding.DecodeString(verifier); err != nil {
		t.Errorf("verifier is not valid base64url: %v", err)
	}
	// Challenge must equal BASE64URL(SHA256(verifier)).
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge != want {
		t.Errorf("challenge mismatch: got %q, want %q", challenge, want)
	}
}

func TestPKCEPairUnique(t *testing.T) {
	// Sanity check: two calls shouldn't produce the same verifier.
	// Not a rigorous entropy test, just catches a broken RNG.
	seen := map[string]bool{}
	for i := 0; i < 8; i++ {
		v, _, err := newPKCEPair()
		if err != nil {
			t.Fatal(err)
		}
		if seen[v] {
			t.Fatalf("duplicate verifier on iteration %d", i)
		}
		seen[v] = true
	}
}

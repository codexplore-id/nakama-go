package nakama

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

// makeJWT builds a minimal unsigned JWT with the supplied claims, suitable
// for unit-testing the session parsing logic.
func makeJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	body, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	return header + "." + base64.RawURLEncoding.EncodeToString(body) + ".sig"
}

func TestSessionParsesJWTClaims(t *testing.T) {
	exp := time.Now().Add(1 * time.Hour).Unix()
	auth := makeJWT(t, map[string]any{
		"exp": exp,
		"iat": time.Now().Unix(),
		"uid": "user-123",
		"usn": "alice",
		"vrs": map[string]any{"k": "v"},
	})
	refresh := makeJWT(t, map[string]any{"exp": time.Now().Add(24 * time.Hour).Unix()})

	s, err := NewSession(auth, refresh, true)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if s.UserId() != "user-123" {
		t.Errorf("UserId: got %q want user-123", s.UserId())
	}
	if s.Username() != "alice" {
		t.Errorf("Username: got %q want alice", s.Username())
	}
	if !s.Created() {
		t.Errorf("Created: got false want true")
	}
	if s.IsExpired() {
		t.Errorf("IsExpired: got true want false")
	}
	if v := s.Vars()["k"]; v != "v" {
		t.Errorf("Vars[k]: got %q want v", v)
	}
}

func TestSessionExpiry(t *testing.T) {
	auth := makeJWT(t, map[string]any{"exp": time.Now().Add(-1 * time.Hour).Unix(), "uid": "u"})
	s, err := NewSession(auth, "", false)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if !s.IsExpired() {
		t.Errorf("IsExpired: expected true for past-expiry token")
	}
}

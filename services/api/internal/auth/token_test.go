package auth

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func testTokenService() *TokenService {
	s := NewTokenService(strings.Repeat("s", 32), strings.Repeat("p", 32), "issuer", "audience", 15*time.Minute, 24*time.Hour)
	s.now = func() time.Time { return time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC) }
	return s
}

func TestAccessTokenValidation(t *testing.T) {
	s := testTokenService()
	userID, sessionID := newUUID(), newUUID()
	raw, err := s.NewAccessToken(userID, "BRAND", sessionID)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := s.ParseAccessToken(raw)
	if err != nil {
		t.Fatalf("parse valid token: %v", err)
	}
	if claims.Subject != userID || claims.SessionID != sessionID || claims.Role != "BRAND" {
		t.Fatalf("unexpected claims: %#v", claims)
	}

	parts := strings.Split(raw, ".")
	parts[1] = parts[1][:len(parts[1])-1] + "A"
	if _, err = s.ParseAccessToken(strings.Join(parts, ".")); err != ErrAccessInvalid {
		t.Fatalf("tampered token error = %v", err)
	}
	if _, err = s.ParseAccessToken("malformed"); err != ErrAccessInvalid {
		t.Fatalf("malformed token error = %v", err)
	}
}

func TestAccessTokenRejectsExpiredIssuerAudienceAndAlgorithm(t *testing.T) {
	base := testTokenService()
	userID, sessionID := newUUID(), newUUID()
	raw, _ := base.NewAccessToken(userID, "CLIPPER", sessionID)

	base.now = func() time.Time { return time.Date(2026, 7, 11, 10, 16, 0, 0, time.UTC) }
	if _, err := base.ParseAccessToken(raw); err != ErrAccessExpired {
		t.Fatalf("expired error = %v", err)
	}

	wrongIssuer := testTokenService()
	wrongIssuer.issuer = "other"
	if _, err := wrongIssuer.ParseAccessToken(raw); err != ErrAccessInvalid {
		t.Fatalf("issuer error = %v", err)
	}
	wrongAudience := testTokenService()
	wrongAudience.audience = "other"
	if _, err := wrongAudience.ParseAccessToken(raw); err != ErrAccessInvalid {
		t.Fatalf("audience error = %v", err)
	}

	parts := strings.Split(raw, ".")
	parts[0] = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	if _, err := testTokenService().ParseAccessToken(strings.Join(parts, ".")); err != ErrAccessInvalid {
		t.Fatalf("algorithm error = %v", err)
	}
}

func TestRefreshTokensAreRandomAndHashed(t *testing.T) {
	s := testTokenService()
	raw1, hash1, _, err := s.NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	raw2, hash2, _, err := s.NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw1 == raw2 || hash1 == hash2 {
		t.Fatal("refresh tokens must be unique")
	}
	if raw1 == hash1 || strings.Contains(hash1, raw1) {
		t.Fatal("stored value exposes raw token")
	}
	if hash1 != s.HashRefreshToken(raw1) {
		t.Fatal("refresh hashing is not deterministic")
	}
}

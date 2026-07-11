package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type TokenService struct {
	secret     []byte
	pepper     []byte
	issuer     string
	audience   string
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

func NewTokenService(secret, pepper, issuer, audience string, accessTTL, refreshTTL time.Duration) *TokenService {
	return &TokenService{[]byte(secret), []byte(pepper), issuer, audience, accessTTL, refreshTTL, time.Now}
}

func (s *TokenService) NewAccessToken(userID, role, sessionID string) (string, error) {
	now := s.now().UTC()
	claims := Claims{
		Subject: userID, Role: role, SessionID: sessionID, JWTID: newUUID(),
		Issuer: s.issuer, Audience: s.audience, IssuedAt: now.Unix(), ExpiresAt: now.Add(s.accessTTL).Unix(),
	}
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	unsigned := encodeSegment(header) + "." + encodeSegment(payload)
	return unsigned + "." + encodeSegment(s.sign(unsigned)), nil
}

func (s *TokenService) ParseAccessToken(raw string) (Claims, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return Claims{}, ErrAccessInvalid
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrAccessInvalid
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if json.Unmarshal(headerBytes, &header) != nil || header.Alg != "HS256" || header.Typ != "JWT" {
		return Claims{}, ErrAccessInvalid
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(signature, s.sign(parts[0]+"."+parts[1])) {
		return Claims{}, ErrAccessInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrAccessInvalid
	}
	var claims Claims
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&claims) != nil {
		return Claims{}, ErrAccessInvalid
	}
	if claims.Subject == "" || claims.Role == "" || claims.SessionID == "" || claims.JWTID == "" ||
		claims.Issuer != s.issuer || claims.Audience != s.audience || claims.IssuedAt <= 0 || claims.ExpiresAt <= 0 ||
		!validUUID(claims.Subject) || !validUUID(claims.SessionID) || !validUUID(claims.JWTID) ||
		!ValidRole(claims.Role) {
		return Claims{}, ErrAccessInvalid
	}
	now := s.now().Unix()
	if claims.ExpiresAt <= now {
		return Claims{}, ErrAccessExpired
	}
	if claims.IssuedAt > now+60 || claims.ExpiresAt <= claims.IssuedAt {
		return Claims{}, ErrAccessInvalid
	}
	return claims, nil
}

func (s *TokenService) NewRefreshToken() (raw, hash string, expiresAt time.Time, err error) {
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(bytes)
	return raw, s.HashRefreshToken(raw), s.now().UTC().Add(s.refreshTTL), nil
}

func (s *TokenService) HashRefreshToken(raw string) string {
	mac := hmac.New(sha256.New, s.pepper)
	_, _ = mac.Write([]byte(raw))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *TokenService) sign(data string) []byte {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(data))
	return mac.Sum(nil)
}

func encodeSegment(data []byte) string { return base64.RawURLEncoding.EncodeToString(data) }

func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("cryptographic randomness unavailable")
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func validUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	_, err := hex.DecodeString(strings.ReplaceAll(value, "-", ""))
	return err == nil
}

var errNotFound = errors.New("not found")

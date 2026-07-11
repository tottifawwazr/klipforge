package auth

import (
	"context"
	"errors"
	"strings"
	"time"
)

type Service struct {
	repository *Repository
	passwords  PasswordService
	tokens     *TokenService
}

func NewService(repository *Repository, passwords PasswordService, tokens *TokenService) *Service {
	return &Service{repository: repository, passwords: passwords, tokens: tokens}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (TokenPair, error) {
	email := NormalizeEmail(input.Email)
	name := strings.TrimSpace(input.FullName)
	role := strings.ToUpper(strings.TrimSpace(input.Role))
	if !ValidateEmail(email) || !ValidatePassword(input.Password) || name == "" || len(name) > 120 || (role != RoleBrand && role != RoleClipper) {
		return TokenPair{}, ErrValidation
	}
	hash, err := s.passwords.Hash(input.Password)
	if err != nil {
		return TokenPair{}, err
	}
	user := User{ID: newUUID(), Email: email, Role: role, IsActive: true, DisplayName: name, CreatedAt: time.Now().UTC()}
	sessionID, familyID, tokenID := newUUID(), newUUID(), newUUID()
	raw, tokenHash, expires, err := s.tokens.NewRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	record := RefreshRecord{ID: tokenID, UserID: user.ID, SessionID: sessionID, FamilyID: familyID, TokenHash: tokenHash, ExpiresAt: expires, UserAgent: input.UserAgent, IPAddress: input.IPAddress}
	audit := AuditEvent{ActorUserID: &user.ID, ActorEmail: user.Email, ActorRole: user.Role, Action: "auth.register", EntityType: "user", EntityID: &user.ID, RequestID: input.RequestID, Metadata: map[string]any{"session_id": sessionID}, IPAddress: input.IPAddress, UserAgent: input.UserAgent}
	if err = s.repository.Register(ctx, user, hash, record, audit); err != nil {
		return TokenPair{}, err
	}
	access, err := s.tokens.NewAccessToken(user.ID, user.Role, sessionID)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{access, raw, expires, user, Session{sessionID, expires}}, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (TokenPair, error) {
	email := NormalizeEmail(input.Email)
	if email == "" || input.Password == "" {
		return TokenPair{}, ErrAuthentication
	}
	user, err := s.repository.UserByEmail(ctx, email)
	passwordHash := user.PasswordHash
	if err != nil {
		passwordHash = "$2a$12$MK/t5MqTGVdxP8k0dLd6Hu.zP9dyAEyfPbyfZHyXu5GUXLAQjYH4y"
	}
	passwordValid := s.passwords.Verify(passwordHash, input.Password)
	if err != nil || !passwordValid {
		audit := AuditEvent{ActorEmail: email, ActorRole: "UNKNOWN", Action: "auth.login_failed", EntityType: "authentication", RequestID: input.RequestID, Metadata: map[string]any{"reason": "invalid_credentials"}, IPAddress: input.IPAddress, UserAgent: input.UserAgent}
		if err == nil {
			audit.ActorUserID = &user.ID
			audit.ActorRole = user.Role
		}
		_ = s.repository.Audit(ctx, audit)
		return TokenPair{}, ErrAuthentication
	}
	if !user.IsActive {
		_ = s.repository.Audit(ctx, AuditEvent{ActorUserID: &user.ID, ActorEmail: user.Email, ActorRole: user.Role, Action: "auth.login_failed", EntityType: "authentication", RequestID: input.RequestID, Metadata: map[string]any{"reason": "user_inactive"}, IPAddress: input.IPAddress, UserAgent: input.UserAgent})
		return TokenPair{}, ErrUserInactive
	}
	sessionID, familyID, tokenID := newUUID(), newUUID(), newUUID()
	raw, hash, expires, err := s.tokens.NewRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	record := RefreshRecord{ID: tokenID, UserID: user.ID, SessionID: sessionID, FamilyID: familyID, TokenHash: hash, ExpiresAt: expires, UserAgent: input.UserAgent, IPAddress: input.IPAddress}
	audit := AuditEvent{ActorUserID: &user.ID, ActorEmail: user.Email, ActorRole: user.Role, Action: "auth.login", EntityType: "session", EntityID: &sessionID, RequestID: input.RequestID, Metadata: map[string]any{}, IPAddress: input.IPAddress, UserAgent: input.UserAgent}
	if err = s.repository.CreateSession(ctx, record, audit); err != nil {
		return TokenPair{}, err
	}
	access, err := s.tokens.NewAccessToken(user.ID, user.Role, sessionID)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{access, raw, expires, user, Session{sessionID, expires}}, nil
}

func (s *Service) Refresh(ctx context.Context, raw, requestID, userAgent, ip string) (TokenPair, error) {
	if strings.TrimSpace(raw) == "" {
		return TokenPair{}, ErrRefreshMissing
	}
	newRaw, newHash, expires, err := s.tokens.NewRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	replacement := RefreshRecord{ID: newUUID(), TokenHash: newHash, ExpiresAt: expires, UserAgent: userAgent, IPAddress: ip}
	audit := AuditEvent{Action: "auth.refresh", EntityType: "session", RequestID: requestID, Metadata: map[string]any{}, IPAddress: ip, UserAgent: userAgent}
	result, err := s.repository.Rotate(ctx, s.tokens.HashRefreshToken(raw), replacement, audit)
	if err != nil {
		return TokenPair{}, err
	}
	access, err := s.tokens.NewAccessToken(result.User.ID, result.User.Role, result.SessionID)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{access, newRaw, expires, result.User, Session{result.SessionID, expires}}, nil
}

func (s *Service) Authenticate(ctx context.Context, raw string) (Principal, error) {
	claims, err := s.tokens.ParseAccessToken(raw)
	if err != nil {
		return Principal{}, err
	}
	user, err := s.repository.UserBySession(ctx, claims.Subject, claims.SessionID)
	if errors.Is(err, errNotFound) {
		return Principal{}, ErrSessionRevoked
	}
	if err != nil {
		return Principal{}, err
	}
	if !user.IsActive {
		return Principal{}, ErrUserInactive
	}
	if user.Role != claims.Role {
		return Principal{}, ErrAccessInvalid
	}
	return Principal{
		UserID: user.ID, Email: user.Email, Role: user.Role, SessionID: claims.SessionID,
		TokenID: claims.JWTID, AccountActive: true, SessionActive: true,
	}, nil
}

func (s *Service) CurrentUser(ctx context.Context, p Principal) (User, error) {
	user, err := s.repository.UserBySession(ctx, p.UserID, p.SessionID)
	if errors.Is(err, errNotFound) {
		return User{}, ErrSessionRevoked
	}
	return user, err
}

func (s *Service) Logout(ctx context.Context, raw, requestID, userAgent, ip string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	audit := AuditEvent{Action: "auth.logout", EntityType: "session", RequestID: requestID, Metadata: map[string]any{}, IPAddress: ip, UserAgent: userAgent}
	return s.repository.RevokeByToken(ctx, s.tokens.HashRefreshToken(raw), audit)
}

func (s *Service) LogoutAll(ctx context.Context, p Principal, requestID, userAgent, ip string) error {
	audit := AuditEvent{Action: "auth.logout_all", EntityType: "user", RequestID: requestID, Metadata: map[string]any{}, IPAddress: ip, UserAgent: userAgent}
	return s.repository.RevokeAll(ctx, p, audit)
}

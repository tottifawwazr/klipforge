package auth

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
	DisplayName  string    `json:"display_name"`
	AvatarURL    *string   `json:"avatar_url"`
	Bio          *string   `json:"bio"`
	CreatedAt    time.Time `json:"created_at"`
	PasswordHash string    `json:"-"`
}

type Session struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	RefreshExpiresAt time.Time
	User             User
	Session          Session
}

type RegisterInput struct {
	Email     string
	Password  string
	FullName  string
	Role      string
	RequestID string
	UserAgent string
	IPAddress string
}

type LoginInput struct {
	Email     string
	Password  string
	RequestID string
	UserAgent string
	IPAddress string
}

type Claims struct {
	Subject   string `json:"sub"`
	Role      string `json:"role"`
	SessionID string `json:"session_id"`
	JWTID     string `json:"jti"`
	Issuer    string `json:"iss"`
	Audience  string `json:"aud"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type Principal struct {
	UserID    string
	Email     string
	Role      string
	SessionID string
}

type AuditEvent struct {
	ActorUserID *string
	ActorEmail  string
	ActorRole   string
	Action      string
	EntityType  string
	EntityID    *string
	RequestID   string
	Metadata    map[string]any
	IPAddress   string
	UserAgent   string
}

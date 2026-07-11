package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

type RefreshRecord struct {
	ID, UserID, SessionID, FamilyID, TokenHash string
	ExpiresAt                                  time.Time
	UserAgent, IPAddress                       string
}

func (r *Repository) Register(ctx context.Context, user User, passwordHash string, token RefreshRecord, audit AuditEvent) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO users (id,email,password_hash,role) VALUES ($1,$2,$3,$4)`, user.ID, user.Email, passwordHash, user.Role)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrEmailRegistered
		}
		return fmt.Errorf("insert user: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO profiles (user_id,display_name) VALUES ($1,$2)`, user.ID, user.DisplayName); err != nil {
		return fmt.Errorf("insert profile: %w", err)
	}
	if err = insertRefresh(ctx, tx, token); err != nil {
		return err
	}
	if err = insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) UserByEmail(ctx context.Context, email string) (User, error) {
	return scanUser(r.pool.QueryRow(ctx, `
		SELECT u.id,u.email,u.role,u.is_active,u.created_at,u.password_hash,p.display_name,p.avatar_url,p.bio
		FROM users u JOIN profiles p ON p.user_id=u.id WHERE u.email=$1`, email))
}

func (r *Repository) UserBySession(ctx context.Context, userID, sessionID string) (User, error) {
	return scanUser(r.pool.QueryRow(ctx, `
		SELECT u.id,u.email,u.role,u.is_active,u.created_at,u.password_hash,p.display_name,p.avatar_url,p.bio
		FROM users u JOIN profiles p ON p.user_id=u.id
		WHERE u.id=$1 AND EXISTS (
			SELECT 1 FROM refresh_tokens rt WHERE rt.user_id=u.id AND rt.session_id=$2
			AND rt.revoked_at IS NULL AND rt.expires_at > NOW())`, userID, sessionID))
}

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Role, &u.IsActive, &u.CreatedAt, &u.PasswordHash, &u.DisplayName, &u.AvatarURL, &u.Bio)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, errNotFound
	}
	return u, err
}

func (r *Repository) CreateSession(ctx context.Context, token RefreshRecord, audit AuditEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = insertRefresh(ctx, tx, token); err != nil {
		return err
	}
	if err = insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func insertRefresh(ctx context.Context, tx pgx.Tx, t RefreshRecord) error {
	_, err := tx.Exec(ctx, `INSERT INTO refresh_tokens
		(id,user_id,session_id,family_id,token_hash,issued_at,expires_at,user_agent,ip_address)
		VALUES ($1,$2,$3,$4,$5,NOW(),$6,NULLIF($7,''),NULLIF($8,'')::inet)`,
		t.ID, t.UserID, t.SessionID, t.FamilyID, t.TokenHash, t.ExpiresAt, t.UserAgent, t.IPAddress)
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

type RotationResult struct {
	User                User
	SessionID, FamilyID string
}

func (r *Repository) Rotate(ctx context.Context, tokenHash string, replacement RefreshRecord, audit AuditEvent) (RotationResult, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return RotationResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var old RefreshRecord
	var revokedAt, rotatedAt *time.Time
	var replacedBy *string
	var user User
	err = tx.QueryRow(ctx, `SELECT rt.id,rt.user_id,rt.session_id,rt.family_id,rt.token_hash,rt.expires_at,
		rt.revoked_at,rt.rotated_at,rt.replaced_by_token_id,
		u.email,u.role,u.is_active,u.created_at,u.password_hash,p.display_name,p.avatar_url,p.bio
		FROM refresh_tokens rt JOIN users u ON u.id=rt.user_id JOIN profiles p ON p.user_id=u.id
		WHERE rt.token_hash=$1 FOR UPDATE OF rt`, tokenHash).Scan(
		&old.ID, &old.UserID, &old.SessionID, &old.FamilyID, &old.TokenHash, &old.ExpiresAt,
		&revokedAt, &rotatedAt, &replacedBy, &user.Email, &user.Role, &user.IsActive, &user.CreatedAt, &user.PasswordHash, &user.DisplayName, &user.AvatarURL, &user.Bio)
	if errors.Is(err, pgx.ErrNoRows) {
		return RotationResult{}, ErrRefreshInvalid
	}
	if err != nil {
		return RotationResult{}, fmt.Errorf("lock refresh token: %w", err)
	}
	user.ID = old.UserID
	if revokedAt != nil || rotatedAt != nil || replacedBy != nil {
		if _, err = tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=COALESCE(revoked_at,NOW()),revocation_reason='REUSE_DETECTED' WHERE session_id=$1`, old.SessionID); err != nil {
			return RotationResult{}, err
		}
		audit.ActorUserID = &old.UserID
		audit.Action = "auth.refresh_reuse"
		audit.ActorEmail = user.Email
		audit.ActorRole = user.Role
		audit.EntityID = &old.SessionID
		if err = insertAudit(ctx, tx, audit); err != nil {
			return RotationResult{}, err
		}
		revocationAudit := audit
		revocationAudit.Action = "auth.session_revoked"
		revocationAudit.Metadata = map[string]any{"reason": "refresh_token_reuse"}
		if err = insertAudit(ctx, tx, revocationAudit); err != nil {
			return RotationResult{}, err
		}
		if err = tx.Commit(ctx); err != nil {
			return RotationResult{}, err
		}
		return RotationResult{}, ErrRefreshReused
	}
	if time.Now().UTC().After(old.ExpiresAt) {
		_, _ = tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=NOW(),revocation_reason='EXPIRED' WHERE id=$1`, old.ID)
		if err = tx.Commit(ctx); err != nil {
			return RotationResult{}, err
		}
		return RotationResult{}, ErrRefreshExpired
	}
	if !user.IsActive {
		_, _ = tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=NOW(),revocation_reason='USER_INACTIVE' WHERE session_id=$1 AND revoked_at IS NULL`, old.SessionID)
		if err = tx.Commit(ctx); err != nil {
			return RotationResult{}, err
		}
		return RotationResult{}, ErrUserInactive
	}
	replacement.UserID = old.UserID
	replacement.SessionID = old.SessionID
	replacement.FamilyID = old.FamilyID
	if err = insertRefresh(ctx, tx, replacement); err != nil {
		return RotationResult{}, err
	}
	command, err := tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=NOW(),rotated_at=NOW(),replaced_by_token_id=$2,revocation_reason='ROTATED' WHERE id=$1 AND revoked_at IS NULL AND rotated_at IS NULL`, old.ID, replacement.ID)
	if err != nil || command.RowsAffected() != 1 {
		return RotationResult{}, fmt.Errorf("rotate refresh token atomically: %w", err)
	}
	audit.ActorUserID = &old.UserID
	audit.ActorEmail = user.Email
	audit.ActorRole = user.Role
	audit.EntityID = &old.SessionID
	if err = insertAudit(ctx, tx, audit); err != nil {
		return RotationResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return RotationResult{}, err
	}
	return RotationResult{user, old.SessionID, old.FamilyID}, nil
}

func (r *Repository) RevokeByToken(ctx context.Context, hash string, audit AuditEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var userID, sessionID, email, role string
	err = tx.QueryRow(ctx, `SELECT rt.user_id,rt.session_id,u.email,u.role FROM refresh_tokens rt JOIN users u ON u.id=rt.user_id WHERE rt.token_hash=$1 FOR UPDATE OF rt`, hash).Scan(&userID, &sessionID, &email, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=COALESCE(revoked_at,NOW()),revocation_reason=COALESCE(revocation_reason,'LOGOUT') WHERE session_id=$1`, sessionID)
	if err != nil {
		return err
	}
	audit.ActorUserID = &userID
	audit.ActorEmail = email
	audit.ActorRole = role
	audit.EntityID = &sessionID
	if err = insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) RevokeAll(ctx context.Context, principal Principal, audit AuditEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=COALESCE(revoked_at,NOW()),revocation_reason=COALESCE(revocation_reason,'LOGOUT_ALL') WHERE user_id=$1`, principal.UserID)
	if err != nil {
		return err
	}
	audit.ActorUserID = &principal.UserID
	audit.ActorEmail = principal.Email
	audit.ActorRole = principal.Role
	audit.EntityID = &principal.UserID
	if err = insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) Audit(ctx context.Context, audit AuditEvent) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func insertAudit(ctx context.Context, tx pgx.Tx, a AuditEvent) error {
	metadata, err := json.Marshal(a.Metadata)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_logs(actor_user_id,actor_email,actor_role,action,entity_type,entity_id,request_id,metadata,ip_address,user_agent)
		VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,NULLIF($9,'')::inet,NULLIF($10,''))`, a.ActorUserID, a.ActorEmail, a.ActorRole, a.Action, a.EntityType, a.EntityID, a.RequestID, metadata, a.IPAddress, a.UserAgent)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func integrationService(t *testing.T) (*Service, *TokenService, *pgxpool.Pool, context.Context) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	tokens := NewTokenService("integration-jwt-secret-at-least-thirty-two-characters", "integration-refresh-pepper-at-least-thirty-two-characters", "test-issuer", "test-audience", 15*time.Minute, time.Hour)
	return NewService(NewRepository(pool), NewPasswordService(bcrypt.MinCost), tokens), tokens, pool, ctx
}

func uniqueEmail(prefix string) string { return fmt.Sprintf("%s-%s@example.test", prefix, newUUID()) }

func cleanupAuthUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, email string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM audit_logs WHERE actor_email=$1`, email)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE email=$1`, email)
	})
}

func registerTestUser(t *testing.T, service *Service, ctx context.Context, email, role string) TokenPair {
	t.Helper()
	pair, err := service.Register(ctx, RegisterInput{Email: email, Password: "StrongPass1", FullName: "Integration User", Role: role, RequestID: "integration-register"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return pair
}

func TestIntegrationRegistrationAndLogin(t *testing.T) {
	service, _, pool, ctx := integrationService(t)
	for _, role := range []string{"BRAND", "CLIPPER"} {
		t.Run(role, func(t *testing.T) {
			email := uniqueEmail("register")
			cleanupAuthUser(t, ctx, pool, email)
			pair := registerTestUser(t, service, ctx, "  "+email[:len(email)-4]+"TEST  ", role)
			if pair.User.Email != email || pair.User.Role != role || pair.AccessToken == "" || pair.RefreshToken == "" {
				t.Fatalf("unexpected registration: %#v", pair)
			}
			var passwordHash, storedToken string
			var profileCount int
			if err := pool.QueryRow(ctx, `SELECT u.password_hash,(SELECT token_hash FROM refresh_tokens WHERE user_id=u.id LIMIT 1),(SELECT count(*) FROM profiles WHERE user_id=u.id) FROM users u WHERE email=$1`, email).Scan(&passwordHash, &storedToken, &profileCount); err != nil {
				t.Fatal(err)
			}
			if passwordHash == "StrongPass1" || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("StrongPass1")) != nil {
				t.Fatal("password was not stored as bcrypt")
			}
			if storedToken == pair.RefreshToken || profileCount != 1 {
				t.Fatal("token hash/profile transaction invariant failed")
			}
			if _, err := service.Register(ctx, RegisterInput{Email: email, Password: "StrongPass1", FullName: "Duplicate", Role: role}); err != ErrEmailRegistered {
				t.Fatalf("duplicate error=%v", err)
			}
		})
	}
	if _, err := service.Register(ctx, RegisterInput{Email: uniqueEmail("admin"), Password: "StrongPass1", FullName: "Admin", Role: "ADMIN"}); err != ErrValidation {
		t.Fatalf("public admin error=%v", err)
	}

	email := uniqueEmail("login")
	cleanupAuthUser(t, ctx, pool, email)
	registerTestUser(t, service, ctx, email, "BRAND")
	login, err := service.Login(ctx, LoginInput{Email: email, Password: "StrongPass1", RequestID: "login"})
	if err != nil || login.AccessToken == "" {
		t.Fatalf("login=%#v err=%v", login, err)
	}
	_, wrong := service.Login(ctx, LoginInput{Email: email, Password: "WrongPass1"})
	unknownEmail := uniqueEmail("unknown")
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM audit_logs WHERE actor_email=$1`, unknownEmail)
	})
	_, unknown := service.Login(ctx, LoginInput{Email: unknownEmail, Password: "WrongPass1"})
	if wrong != ErrAuthentication || unknown != ErrAuthentication || AsError(wrong).Message != AsError(unknown).Message {
		t.Fatalf("generic failures differ: %v %v", wrong, unknown)
	}
	if _, err = pool.Exec(ctx, `UPDATE users SET is_active=false WHERE email=$1`, email); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Login(ctx, LoginInput{Email: email, Password: "StrongPass1"}); err != ErrUserInactive {
		t.Fatalf("inactive login=%v", err)
	}
}

func TestIntegrationRotationReuseAndConcurrentRefresh(t *testing.T) {
	service, tokens, pool, ctx := integrationService(t)
	email := uniqueEmail("rotate")
	cleanupAuthUser(t, ctx, pool, email)
	registerTestUser(t, service, ctx, email, "CLIPPER")
	login, err := service.Login(ctx, LoginInput{Email: email, Password: "StrongPass1"})
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := service.Refresh(ctx, login.RefreshToken, "refresh", "agent", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if rotated.RefreshToken == login.RefreshToken || rotated.Session.ID != login.Session.ID {
		t.Fatal("rotation did not preserve session with a new token")
	}
	if _, err = service.Refresh(ctx, login.RefreshToken, "reuse", "agent", "127.0.0.1"); err != ErrRefreshReused {
		t.Fatalf("reuse error=%v", err)
	}
	if _, err = service.Authenticate(ctx, rotated.AccessToken); err != ErrSessionRevoked {
		t.Fatalf("reused family remained active: %v", err)
	}
	var auditCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE actor_email=$1 AND action='auth.refresh_reuse'`, email).Scan(&auditCount); err != nil || auditCount < 1 {
		t.Fatalf("reuse audit count=%d err=%v", auditCount, err)
	}

	concurrent, err := service.Login(ctx, LoginInput{Email: email, Password: "StrongPass1"})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(2)
	for i := range errs {
		go func(index int) {
			defer wg.Done()
			_, errs[index] = service.Refresh(context.Background(), concurrent.RefreshToken, "concurrent", "agent", "127.0.0.1")
		}(i)
	}
	wg.Wait()
	var success, reused int
	for _, refreshErr := range errs {
		if refreshErr == nil {
			success++
		}
		if errors.Is(refreshErr, ErrRefreshReused) {
			reused++
		}
	}
	if success != 1 || reused != 1 {
		t.Fatalf("concurrent results=%v", errs)
	}
	var active int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM refresh_tokens WHERE session_id=$1 AND revoked_at IS NULL`, concurrent.Session.ID).Scan(&active); err != nil || active != 0 {
		t.Fatalf("active concurrent replacements=%d err=%v", active, err)
	}
	_ = tokens
}

func TestIntegrationLogoutAndLogoutAll(t *testing.T) {
	service, _, pool, ctx := integrationService(t)
	email := uniqueEmail("logout")
	cleanupAuthUser(t, ctx, pool, email)
	registerTestUser(t, service, ctx, email, "BRAND")
	one, _ := service.Login(ctx, LoginInput{Email: email, Password: "StrongPass1"})
	two, _ := service.Login(ctx, LoginInput{Email: email, Password: "StrongPass1"})
	if err := service.Logout(ctx, one.RefreshToken, "logout", "", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(ctx, one.AccessToken); err != ErrSessionRevoked {
		t.Fatalf("logged-out access token error=%v", err)
	}
	if err := service.Logout(ctx, one.RefreshToken, "logout-again", "", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Refresh(ctx, one.RefreshToken, "", "", ""); err != ErrRefreshReused {
		t.Fatalf("revoked refresh error=%v", err)
	}
	p, err := service.Authenticate(ctx, two.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	three, _ := service.Login(ctx, LoginInput{Email: email, Password: "StrongPass1"})
	if err = service.LogoutAll(ctx, p, "logout-all", "", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Refresh(ctx, two.RefreshToken, "", "", ""); err != ErrRefreshReused {
		t.Fatalf("session two refresh=%v", err)
	}
	if _, err = service.Refresh(ctx, three.RefreshToken, "", "", ""); err != ErrRefreshReused {
		t.Fatalf("session three refresh=%v", err)
	}
}

func TestIntegrationExpiredRefreshTokenIsRejected(t *testing.T) {
	service, tokens, pool, ctx := integrationService(t)
	email := uniqueEmail("expired")
	cleanupAuthUser(t, ctx, pool, email)
	registerTestUser(t, service, ctx, email, "CLIPPER")
	login, err := service.Login(ctx, LoginInput{Email: email, Password: "StrongPass1"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `UPDATE refresh_tokens SET created_at=NOW()-INTERVAL '2 hours',issued_at=NOW()-INTERVAL '2 hours',expires_at=NOW()-INTERVAL '1 hour' WHERE token_hash=$1`, tokens.HashRefreshToken(login.RefreshToken))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Refresh(ctx, login.RefreshToken, "expired", "", ""); err != ErrRefreshExpired {
		t.Fatalf("expired refresh error=%v", err)
	}
}

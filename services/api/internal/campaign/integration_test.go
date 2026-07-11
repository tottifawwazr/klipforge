package campaign

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klipforge/klipforge/services/api/internal/auth"
)

const (
	seedBrandOne = "00000000-0000-0000-0000-000000000101"
	seedBrandTwo = "00000000-0000-0000-0000-000000000102"
	seedClipper  = "00000000-0000-0000-0000-000000000201"
	seedAdmin    = "00000000-0000-0000-0000-000000000001"
)

func campaignIntegration(t *testing.T) (*Service, *Repository, *pgxpool.Pool, context.Context) {
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
	repo := NewRepository(pool)
	return NewService(repo), repo, pool, ctx
}
func campaignActor(id, role, email string) auth.Principal {
	return auth.Principal{UserID: id, Email: email, Role: role, SessionID: "00000000-0000-0000-0000-00000000f101", TokenID: "00000000-0000-0000-0000-00000000f102", AccountActive: true, SessionActive: true}
}
func uniqueSlug(t *testing.T) string { return fmt.Sprintf("campaign-%d", time.Now().UnixNano()) }
func createInput(slug string) CreateInput {
	start := time.Now().UTC().AddDate(0, 0, 2)
	end := start.AddDate(0, 0, 14)
	return CreateInput{Title: "Creator launch campaign", Slug: slug, Description: "A complete campaign description for creator launch work.", Brief: "A concise but complete creator launch brief.", TotalBudget: "1200.00", CPMRate: "15.5000", MaximumPayoutPerClip: "125.00", StartDate: start.Format("2006-01-02"), EndDate: end.Format("2006-01-02"), Platforms: []string{"TIKTOK", "INSTAGRAM"}, Requirements: []Requirement{{Type: "CONTENT", Description: "Show the product in the first five seconds."}}, RequestID: "campaign-integration"}
}
func cleanupCampaign(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM audit_logs WHERE entity_type='campaign' AND entity_id=$1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1`, id)
	})
}

func TestCampaignCreateTransactionAndValidation(t *testing.T) {
	service, repo, pool, ctx := campaignIntegration(t)
	brand := campaignActor(seedBrandOne, auth.RoleBrand, "brand@klipforge.local")
	created, err := service.Create(ctx, brand, createInput(uniqueSlug(t)))
	if err != nil {
		t.Fatal(err)
	}
	cleanupCampaign(t, pool, created.ID)
	if created.BrandID != seedBrandOne || created.Status != StatusDraft || created.RemainingBudget != "1200.00" || len(created.Platforms) != 2 || len(created.Requirements) != 1 {
		t.Fatalf("unexpected campaign: %#v", created)
	}
	var audits int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_id=$1 AND action='campaign.created'`, created.ID).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("create audit=%d err=%v", audits, err)
	}
	if _, err = service.Create(ctx, brand, createInput(created.Slug)); err != ErrSlugConflict {
		t.Fatalf("duplicate slug error=%v", err)
	}
	invalid := createInput(uniqueSlug(t))
	invalid.TotalBudget = "0"
	if _, err = service.Create(ctx, brand, invalid); err != ErrInvalidBudget {
		t.Fatalf("budget error=%v", err)
	}
	invalid = createInput(uniqueSlug(t))
	invalid.Platforms = []string{"VIMEO"}
	if _, err = service.Create(ctx, brand, invalid); err != ErrInvalidPlatform {
		t.Fatalf("platform error=%v", err)
	}
	invalid = createInput(uniqueSlug(t))
	invalid.EndDate = invalid.StartDate
	if _, err = service.Create(ctx, brand, invalid); err != ErrInvalidDateRange {
		t.Fatalf("date error=%v", err)
	}
	if _, err = service.Create(ctx, campaignActor(seedClipper, auth.RoleClipper, "clipper@klipforge.local"), createInput(uniqueSlug(t))); err != auth.ErrRoleNotAllowed {
		t.Fatalf("clipper create error=%v", err)
	}
	failed := Campaign{BrandID: seedBrandOne, Title: "Rollback campaign", Slug: uniqueSlug(t), Description: "A transaction rollback campaign description.", Brief: "A transaction rollback campaign brief.", TotalBudget: "100.00", CPMRate: strPtr("5.00"), MaximumPayoutPerClip: "10.00", StartDate: time.Now().UTC().AddDate(0, 0, 2), EndDate: time.Now().UTC().AddDate(0, 0, 4), Platforms: []string{"INVALID"}}
	if _, err = repo.Create(ctx, failed, brand, "rollback", "", ""); err == nil {
		t.Fatal("expected related insert failure")
	}
	var exists bool
	if err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM campaigns WHERE slug=$1)`, failed.Slug).Scan(&exists); err != nil || exists {
		t.Fatalf("transaction rollback failed exists=%v err=%v", exists, err)
	}
}

func TestCampaignReadsUpdateOwnershipAndListing(t *testing.T) {
	service, _, pool, ctx := campaignIntegration(t)
	brand := campaignActor(seedBrandOne, auth.RoleBrand, "brand@klipforge.local")
	other := campaignActor(seedBrandTwo, auth.RoleBrand, "brand-summit@klipforge.local")
	created, err := service.Create(ctx, brand, createInput(uniqueSlug(t)))
	if err != nil {
		t.Fatal(err)
	}
	cleanupCampaign(t, pool, created.ID)
	if _, err = service.PublicGet(ctx, created.ID); err != ErrNotFound {
		t.Fatalf("draft public read=%v", err)
	}
	if _, err = service.BrandGet(ctx, other, created.ID); err != ErrAccessDenied {
		t.Fatalf("other brand read=%v", err)
	}
	title := "Updated creator campaign"
	platforms := []string{"YOUTUBE"}
	requirements := []Requirement{{Type: "DISCLOSURE", Description: "Use the required disclosure in the caption."}}
	updated, err := service.Update(ctx, brand, created.ID, UpdateInput{Title: &title, Platforms: &platforms, Requirements: &requirements, RequestID: "update"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != title || len(updated.Platforms) != 1 || updated.Platforms[0] != "YOUTUBE" || len(updated.Requirements) != 1 {
		t.Fatalf("update result=%#v", updated)
	}
	if _, err = service.Update(ctx, other, created.ID, UpdateInput{Title: &title}); err != ErrAccessDenied {
		t.Fatalf("other brand update=%v", err)
	}
	items, page, err := service.BrandList(ctx, brand, ListQuery{Page: 1, Limit: 1, Search: "updated", Sort: "title", Direction: "asc"})
	if err != nil || page.Limit != 1 || len(items) == 0 {
		t.Fatalf("brand list items=%d page=%#v err=%v", len(items), page, err)
	}
	if _, _, err = service.BrandList(ctx, brand, ListQuery{Sort: "budget_amount"}); err != ErrInvalidQuery {
		t.Fatalf("sort error=%v", err)
	}
	if _, err = service.BrandGet(ctx, campaignActor(seedClipper, auth.RoleClipper, "clipper@klipforge.local"), created.ID); err != ErrAccessDenied {
		t.Fatalf("clipper private access=%v", err)
	}
}

func TestCampaignLifecycleAndAudit(t *testing.T) {
	service, _, pool, ctx := campaignIntegration(t)
	brand := campaignActor(seedBrandOne, auth.RoleBrand, "brand@klipforge.local")
	created, err := service.Create(ctx, brand, createInput(uniqueSlug(t)))
	if err != nil {
		t.Fatal(err)
	}
	cleanupCampaign(t, pool, created.ID)
	active, err := service.Publish(ctx, brand, created.ID, "publish", "", "")
	if err != nil || active.Status != StatusActive {
		t.Fatalf("publish status=%s err=%v", active.Status, err)
	}
	paused, err := service.Pause(ctx, brand, created.ID, "pause", "", "")
	if err != nil || paused.Status != StatusPaused {
		t.Fatalf("pause status=%s err=%v", paused.Status, err)
	}
	pausedTitle := "Paused campaign update"
	if _, err = service.Update(ctx, brand, created.ID, UpdateInput{Title: &pausedTitle, RequestID: "paused-update"}); err != nil {
		t.Fatalf("paused update: %v", err)
	}
	resumed, err := service.Resume(ctx, brand, created.ID, "resume", "", "")
	if err != nil || resumed.Status != StatusActive {
		t.Fatalf("resume status=%s err=%v", resumed.Status, err)
	}
	completed, err := service.Complete(ctx, brand, created.ID, "complete", "", "")
	if err != nil || completed.Status != StatusCompleted {
		t.Fatalf("complete status=%s err=%v", completed.Status, err)
	}
	if _, err = service.Cancel(ctx, brand, created.ID, "cancel", "", ""); err != ErrAlreadyTerminal {
		t.Fatalf("terminal cancellation=%v", err)
	}
	var audits int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_id=$1 AND action IN ('campaign.published','campaign.paused','campaign.resumed','campaign.completed')`, created.ID).Scan(&audits); err != nil || audits != 4 {
		t.Fatalf("lifecycle audits=%d err=%v", audits, err)
	}
	incomplete := createInput(uniqueSlug(t))
	incomplete.Platforms = nil
	draft, err := service.Create(ctx, brand, incomplete)
	if err != nil {
		t.Fatal(err)
	}
	cleanupCampaign(t, pool, draft.ID)
	if _, err = service.Publish(ctx, brand, draft.ID, "publish", "", ""); err != ErrNotReadyToPublish {
		t.Fatalf("incomplete publish=%v", err)
	}
	cancelled, err := service.Cancel(ctx, brand, draft.ID, "cancel", "", "")
	if err != nil || cancelled.Status != StatusCancelled {
		t.Fatalf("cancel=%#v err=%v", cancelled, err)
	}
	if _, err = service.Publish(ctx, brand, draft.ID, "publish", "", ""); err != ErrAlreadyTerminal {
		t.Fatalf("terminal publish=%v", err)
	}
	adminTarget, err := service.Create(ctx, brand, createInput(uniqueSlug(t)))
	if err != nil {
		t.Fatal(err)
	}
	cleanupCampaign(t, pool, adminTarget.ID)
	if _, err = service.AdminCancel(ctx, campaignActor(seedAdmin, auth.RoleAdmin, "admin@klipforge.local"), adminTarget.ID, "admin-cancel", "", ""); err != nil {
		t.Fatalf("admin cancel explicit action=%v", err)
	}
}

func TestExpiredPausedCampaignCannotResume(t *testing.T) {
	service, _, pool, ctx := campaignIntegration(t)
	brand := campaignActor(seedBrandOne, auth.RoleBrand, "brand@klipforge.local")
	created, err := service.Create(ctx, brand, createInput(uniqueSlug(t)))
	if err != nil {
		t.Fatal(err)
	}
	cleanupCampaign(t, pool, created.ID)
	if _, err = service.Publish(ctx, brand, created.ID, "publish", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Pause(ctx, brand, created.ID, "pause", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE campaigns SET start_date=CURRENT_DATE - 10, end_date=CURRENT_DATE - 1 WHERE id=$1`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Resume(ctx, brand, created.ID, "resume", "", ""); err != ErrNotReadyToPublish {
		t.Fatalf("expired resume=%v", err)
	}
}

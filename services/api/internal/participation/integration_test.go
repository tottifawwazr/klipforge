package participation

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/klipforge/klipforge/services/api/internal/campaign"
)

const (
	participationSeedBrand      = "00000000-0000-0000-0000-000000000101"
	participationOtherBrand     = "00000000-0000-0000-0000-000000000102"
	participationSeedClipper    = "00000000-0000-0000-0000-000000000201"
	participationOtherClipper   = "00000000-0000-0000-0000-000000000202"
	participationSeedAdmin      = "00000000-0000-0000-0000-000000000001"
	participationBrandEmail     = "brand@klipforge.local"
	participationClipperEmail   = "clipper@klipforge.local"
	participationOtherClipperEM = "clipper-beryl@klipforge.local"
)

func participationIntegration(t *testing.T) (*Service, *Repository, *pgxpool.Pool, context.Context) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	repo := NewRepository(pool)
	return NewService(repo, campaign.NewRepository(pool)), repo, pool, ctx
}

func participationActor(id, role, email string) auth.Principal {
	return auth.Principal{UserID: id, Email: email, Role: role, SessionID: "00000000-0000-0000-0000-00000000f201", TokenID: "00000000-0000-0000-0000-00000000f202", AccountActive: true, SessionActive: true}
}

func activeCampaign(t *testing.T, pool *pgxpool.Pool, ctx context.Context) string {
	t.Helper()
	slug := fmt.Sprintf("participation-test-%d", time.Now().UnixNano())
	var id string
	err := pool.QueryRow(ctx, `INSERT INTO campaigns (brand_id,name,slug,description,brief,status,budget_amount,remaining_budget_amount,cpm_amount,max_payout_per_clip,currency_code,start_date,end_date)
		VALUES ($1,$2,$3,$4,$5,'ACTIVE','1000','1000','10','100','USD',CURRENT_DATE - 1,CURRENT_DATE + 14) RETURNING id`,
		participationSeedBrand, "Participation integration campaign", slug, "A campaign used only to verify participation and submission behavior.", "A campaign used only to verify participation behavior.").Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO campaign_platforms (campaign_id,platform) VALUES ($1,'TIKTOK'),($1,'INSTAGRAM'),($1,'YOUTUBE')`, id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM audit_logs WHERE metadata->>'campaign_id'=$1 OR entity_id=$1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM clip_submissions WHERE campaign_id=$1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1`, id)
	})
	return id
}

func TestParticipationAndSubmissionIntegration(t *testing.T) {
	service, _, pool, ctx := participationIntegration(t)
	campaignID := activeCampaign(t, pool, ctx)
	clipper := participationActor(participationSeedClipper, auth.RoleClipper, participationClipperEmail)
	otherClipper := participationActor(participationOtherClipper, auth.RoleClipper, participationOtherClipperEM)
	brand := participationActor(participationSeedBrand, auth.RoleBrand, participationBrandEmail)

	joined, err := service.Join(ctx, clipper, campaignID, "join-test", "127.0.0.1", "test")
	if err != nil || joined.ClipperID != clipper.UserID || joined.Status != "ACCEPTED" {
		t.Fatalf("join = %#v, %v", joined, err)
	}
	if _, err = service.Join(ctx, clipper, campaignID, "duplicate-join", "127.0.0.1", "test"); err != ErrAlreadyJoined {
		t.Fatalf("duplicate join error = %v", err)
	}
	var participationCount, auditCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM campaign_participants WHERE campaign_id=$1 AND clipper_id=$2`, campaignID, clipper.UserID).Scan(&participationCount); err != nil || participationCount != 1 {
		t.Fatalf("participations = %d, err = %v", participationCount, err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_id=$1 AND action='campaign.joined'`, joined.ID).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("join audits = %d, err = %v", auditCount, err)
	}

	submission, err := service.Submit(ctx, clipper, campaignID, SubmitInput{Platform: "TIKTOK", ContentURL: "https://www.tiktok.com/@ari/video/phase3b?utm_source=test", Caption: "A safe caption", RequestID: "submission-test", IPAddress: "127.0.0.1", UserAgent: "test"})
	if err != nil || submission.Status != "PENDING" || submission.ContentURL != "https://tiktok.com/@ari/video/phase3b" {
		t.Fatalf("submission = %#v, %v", submission, err)
	}
	if _, err = service.Submit(ctx, clipper, campaignID, SubmitInput{Platform: "TIKTOK", ContentURL: "https://tiktok.com/@ari/video/phase3b", RequestID: "duplicate-submission"}); err != ErrDuplicateURL {
		t.Fatalf("duplicate URL error = %v", err)
	}
	if _, err = service.Submit(ctx, otherClipper, campaignID, SubmitInput{Platform: "TIKTOK", ContentURL: "https://tiktok.com/@beryl/video/nonparticipant"}); err != ErrSubmissionRequiresParticipation {
		t.Fatalf("non-participant submit error = %v", err)
	}
	if _, err = service.Submit(ctx, clipper, campaignID, SubmitInput{Platform: "VIMEO", ContentURL: "https://vimeo.com/42"}); err != ErrInvalidPlatform {
		t.Fatalf("invalid platform error = %v", err)
	}
	if _, err = service.Submit(ctx, clipper, campaignID, SubmitInput{Platform: "TIKTOK", ContentURL: "https://example.com/42"}); err != ErrInvalidURL {
		t.Fatalf("invalid URL error = %v", err)
	}

	if _, err = service.Join(ctx, otherClipper, campaignID, "other-join", "127.0.0.1", "test"); err != nil {
		t.Fatal(err)
	}
	otherSubmission, err := service.Submit(ctx, otherClipper, campaignID, SubmitInput{Platform: "INSTAGRAM", ContentURL: "https://www.instagram.com/reel/phase3b-other"})
	if err != nil {
		t.Fatal(err)
	}
	items, page, err := service.MySubmissions(ctx, clipper, ListQuery{Page: 1, Limit: 10, CampaignID: campaignID, Platform: "TIKTOK", Status: "PENDING", Sort: "submitted_at"})
	if err != nil || len(items) != 1 || items[0].ID != submission.ID || page.TotalItems != 1 {
		t.Fatalf("my submissions = %#v, page = %#v, err = %v", items, page, err)
	}
	if _, err = service.MySubmission(ctx, clipper, otherSubmission.ID); err != ErrSubmissionNotFound {
		t.Fatalf("cross-clipper submission lookup error = %v", err)
	}
	brandItems, _, err := service.BrandSubmissions(ctx, brand, campaignID, ListQuery{Page: 1, Limit: 10})
	if err != nil || len(brandItems) != 2 {
		t.Fatalf("brand submissions = %d, err = %v", len(brandItems), err)
	}
	if _, _, err = service.BrandSubmissions(ctx, participationActor(participationOtherBrand, auth.RoleBrand, "brand-summit@klipforge.local"), campaignID, ListQuery{}); err != ErrCampaignNotFound {
		t.Fatalf("other brand access error = %v", err)
	}

	platform := "INSTAGRAM"
	if _, err = service.Update(ctx, clipper, submission.ID, UpdateInput{Platform: &platform}); err != ErrInvalidURL {
		t.Fatalf("platform/content mismatch error = %v", err)
	}
	caption := "Updated safe caption"
	updated, err := service.Update(ctx, clipper, submission.ID, UpdateInput{Caption: &caption, RequestID: "update-test", IPAddress: "127.0.0.1", UserAgent: "test"})
	if err != nil || updated.Caption != caption {
		t.Fatalf("update = %#v, %v", updated, err)
	}
	if _, err = service.Update(ctx, clipper, submission.ID, UpdateInput{}); err != ErrInvalidQuery {
		t.Fatalf("empty update error = %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE clip_submissions SET status='APPROVED' WHERE id=$1`, submission.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Update(ctx, clipper, submission.ID, UpdateInput{Caption: &caption}); err != ErrNotEditable {
		t.Fatalf("approved update error = %v", err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_id=$1 AND action IN ('submission.created','submission.updated')`, submission.ID).Scan(&auditCount); err != nil || auditCount != 2 {
		t.Fatalf("submission audits = %d, err = %v", auditCount, err)
	}
}

func TestParticipationTransactionRollsBackWhenAuditFails(t *testing.T) {
	_, repository, pool, ctx := participationIntegration(t)
	campaignID := activeCampaign(t, pool, ctx)
	actor := participationActor(participationSeedClipper, auth.RoleClipper, participationClipperEmail)
	if _, err := repository.Join(ctx, campaignID, actor, "rollback-test", "not-an-ip", "test"); err == nil {
		t.Fatal("expected audit insert failure")
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM campaign_participants WHERE campaign_id=$1 AND clipper_id=$2`, campaignID, actor.UserID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rollback count = %d, err = %v", count, err)
	}
}

func TestAdminInspectionIntegration(t *testing.T) {
	service, _, pool, ctx := participationIntegration(t)
	campaignID := activeCampaign(t, pool, ctx)
	clipper := participationActor(participationSeedClipper, auth.RoleClipper, participationClipperEmail)
	admin := participationActor(participationSeedAdmin, auth.RoleAdmin, "admin@klipforge.local")
	joined, err := service.Join(ctx, clipper, campaignID, "admin-join", "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	submission, err := service.Submit(ctx, clipper, campaignID, SubmitInput{Platform: "YOUTUBE", ContentURL: "https://www.youtube.com/shorts/phase3b-admin"})
	if err != nil {
		t.Fatal(err)
	}
	participations, _, err := service.AdminParticipations(ctx, admin, ListQuery{CampaignID: campaignID})
	if err != nil || len(participations) != 1 || participations[0].ID != joined.ID {
		t.Fatalf("admin participations = %#v, err = %v", participations, err)
	}
	if _, err = service.AdminParticipation(ctx, admin, joined.ID); err != nil {
		t.Fatal(err)
	}
	submissions, _, err := service.AdminSubmissions(ctx, admin, ListQuery{CampaignID: campaignID})
	if err != nil || len(submissions) != 1 || submissions[0].ID != submission.ID {
		t.Fatalf("admin submissions = %#v, err = %v", submissions, err)
	}
	if _, err = service.AdminSubmission(ctx, admin, submission.ID); err != nil {
		t.Fatal(err)
	}
}

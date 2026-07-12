package moderation

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/klipforge/klipforge/services/api/internal/participation"
)

const (
	moderationBrandOne   = "00000000-0000-0000-0000-000000000101"
	moderationBrandTwo   = "00000000-0000-0000-0000-000000000102"
	moderationClipper    = "00000000-0000-0000-0000-000000000201"
	moderationAdmin      = "00000000-0000-0000-0000-000000000001"
	moderationOtherRoute = "00000000-0000-0000-0000-000000000302"
)

func moderationIntegration(t *testing.T) (*Service, *pgxpool.Pool, context.Context) {
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
	return NewService(NewRepository(pool)), pool, ctx
}

func moderationActor(id, role, email string) auth.Principal {
	return auth.Principal{UserID: id, Role: role, Email: email, AccountActive: true, SessionActive: true, SessionID: "00000000-0000-0000-0000-00000000c001", TokenID: "00000000-0000-0000-0000-00000000c002"}
}

type moderationFixture struct {
	campaignID    string
	participantID string
}

func createModerationFixture(t *testing.T, pool *pgxpool.Pool, ctx context.Context) moderationFixture {
	t.Helper()
	suffix := time.Now().UnixNano()
	var fixture moderationFixture
	err := pool.QueryRow(ctx, `INSERT INTO campaigns (brand_id,name,slug,description,brief,status,budget_amount,remaining_budget_amount,cpm_amount,max_payout_per_clip,currency_code,start_date,end_date)
		VALUES ($1,$2,$3,$4,$5,'ACTIVE','1000','1000','10','100','USD',CURRENT_DATE - 1,CURRENT_DATE + 14) RETURNING id`,
		moderationBrandOne, "Moderation integration campaign", fmt.Sprintf("moderation-integration-%d", suffix),
		"A campaign used to verify moderation persistence and ownership.", "A campaign used to verify moderation behavior.").Scan(&fixture.campaignID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO campaign_platforms(campaign_id,platform) VALUES($1,'TIKTOK'),($1,'INSTAGRAM')`, fixture.campaignID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO campaign_requirements(campaign_id,requirement_type,description,position) VALUES($1,'CONTENT','Show the product in the first five seconds.',1)`, fixture.campaignID); err != nil {
		t.Fatal(err)
	}
	err = pool.QueryRow(ctx, `INSERT INTO campaign_participants(campaign_id,clipper_id,status,joined_at) VALUES($1,$2,'ACCEPTED',NOW()) RETURNING id`, fixture.campaignID, moderationClipper).Scan(&fixture.participantID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM audit_logs WHERE metadata->>'campaign_id'=$1`, fixture.campaignID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM clip_submissions WHERE campaign_id=$1`, fixture.campaignID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1`, fixture.campaignID)
	})
	return fixture
}

func createModerationSubmission(t *testing.T, pool *pgxpool.Pool, ctx context.Context, fixture moderationFixture, platform string) string {
	t.Helper()
	var id string
	path := fmt.Sprintf("moderation-%d", time.Now().UnixNano())
	url := "https://tiktok.com/@moderation/video/" + path
	if platform == "INSTAGRAM" {
		url = "https://instagram.com/reel/" + path
	}
	err := pool.QueryRow(ctx, `INSERT INTO clip_submissions(campaign_id,participant_id,content_url,platform,caption,status) VALUES($1,$2,$3,$4,'A moderation integration caption.','PENDING') RETURNING id`, fixture.campaignID, fixture.participantID, url, platform).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestModerationPersistenceOwnershipQueuesAndAudit(t *testing.T) {
	service, pool, ctx := moderationIntegration(t)
	fixture := createModerationFixture(t, pool, ctx)
	brand := moderationActor(moderationBrandOne, auth.RoleBrand, "brand@klipforge.local")
	otherBrand := moderationActor(moderationBrandTwo, auth.RoleBrand, "brand-summit@klipforge.local")
	admin := moderationActor(moderationAdmin, auth.RoleAdmin, "admin@klipforge.local")
	clipper := moderationActor(moderationClipper, auth.RoleClipper, "clipper@klipforge.local")

	approvedID := createModerationSubmission(t, pool, ctx, fixture, "TIKTOK")
	approved, err := service.BrandApprove(ctx, brand, fixture.campaignID, approvedID, ActionInput{RequestID: "approve", IPAddress: "127.0.0.1", UserAgent: "test"})
	if err != nil || approved.Status != StatusApproved || approved.ReviewedByUserID == nil || *approved.ReviewedByUserID != brand.UserID || approved.ReviewedAt == nil || approved.Reason != nil {
		t.Fatalf("approved = %#v, err = %v", approved, err)
	}
	if _, err = service.BrandApprove(ctx, brand, fixture.campaignID, approvedID, ActionInput{}); err != ErrAlreadyReviewed {
		t.Fatalf("repeat approval error = %v", err)
	}

	rejectedID := createModerationSubmission(t, pool, ctx, fixture, "INSTAGRAM")
	rejected, err := service.BrandReject(ctx, brand, fixture.campaignID, rejectedID, ActionInput{Reason: "  Does not follow the campaign brief.  ", RequestID: "reject", IPAddress: "127.0.0.1"})
	if err != nil || rejected.Status != StatusRejected || rejected.Reason == nil || *rejected.Reason != "Does not follow the campaign brief." || rejected.ReviewedAt == nil {
		t.Fatalf("rejected = %#v, err = %v", rejected, err)
	}
	clipperView, _, _, err := participation.NewRepository(pool).Submission(ctx, rejectedID)
	if err != nil || clipperView.RejectionReason == nil || *clipperView.RejectionReason != "Does not follow the campaign brief." {
		t.Fatalf("clipper rejection view = %#v, err = %v", clipperView, err)
	}

	resolvedID := createModerationSubmission(t, pool, ctx, fixture, "TIKTOK")
	flagged, err := service.BrandFlag(ctx, brand, fixture.campaignID, resolvedID, ActionInput{Reason: "Possible policy issue.", RequestID: "flag", IPAddress: "127.0.0.1"})
	if err != nil || flagged.Status != StatusFlagged || flagged.Reason == nil {
		t.Fatalf("flagged = %#v, err = %v", flagged, err)
	}
	clipperFlagView, _, _, err := participation.NewRepository(pool).Submission(ctx, resolvedID)
	if err != nil || clipperFlagView.RejectionReason != nil {
		t.Fatalf("flag reason leaked to clipper view = %#v, err = %v", clipperFlagView, err)
	}
	resolved, err := service.BrandApprove(ctx, brand, fixture.campaignID, resolvedID, ActionInput{RequestID: "resolve", IPAddress: "127.0.0.1"})
	if err != nil || resolved.Status != StatusApproved || resolved.Reason != nil {
		t.Fatalf("resolved = %#v, err = %v", resolved, err)
	}

	adminResolvedID := createModerationSubmission(t, pool, ctx, fixture, "INSTAGRAM")
	if _, err = service.AdminFlag(ctx, admin, adminResolvedID, ActionInput{Reason: "Administrative review required.", RequestID: "admin-flag", IPAddress: "127.0.0.1"}); err != nil {
		t.Fatal(err)
	}
	adminResolved, err := service.AdminReject(ctx, admin, adminResolvedID, ActionInput{Reason: "Confirmed policy violation.", RequestID: "admin-resolve", IPAddress: "127.0.0.1"})
	if err != nil || adminResolved.Status != StatusRejected || adminResolved.ReviewedByUserID == nil || *adminResolved.ReviewedByUserID != admin.UserID {
		t.Fatalf("admin resolution = %#v, err = %v", adminResolved, err)
	}

	protectedID := createModerationSubmission(t, pool, ctx, fixture, "TIKTOK")
	if _, err = service.BrandApprove(ctx, otherBrand, fixture.campaignID, protectedID, ActionInput{}); err != ErrSubmissionNotFound {
		t.Fatalf("unrelated brand error = %v", err)
	}
	if _, err = service.BrandApprove(ctx, brand, moderationOtherRoute, protectedID, ActionInput{}); err != ErrSubmissionNotFound {
		t.Fatalf("route mismatch error = %v", err)
	}
	if _, err = service.BrandApprove(ctx, clipper, fixture.campaignID, protectedID, ActionInput{}); err != auth.ErrRoleNotAllowed {
		t.Fatalf("clipper moderation error = %v", err)
	}

	openFlagID := createModerationSubmission(t, pool, ctx, fixture, "INSTAGRAM")
	if _, err = service.BrandFlag(ctx, brand, fixture.campaignID, openFlagID, ActionInput{Reason: "Needs another look.", RequestID: "open-flag", IPAddress: "127.0.0.1"}); err != nil {
		t.Fatal(err)
	}
	queue, page, err := service.BrandQueue(ctx, brand, fixture.campaignID, QueueQuery{Page: 1, Limit: 10})
	if err != nil || len(queue) != 2 || page.TotalItems != 2 {
		t.Fatalf("default queue = %d, page=%#v, err=%v", len(queue), page, err)
	}
	filtered, _, err := service.BrandQueue(ctx, brand, fixture.campaignID, QueueQuery{Status: StatusFlagged, Platform: "INSTAGRAM", Search: "Ari", SubmittedFrom: time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02"), Sort: "reviewed_at", Direction: "desc"})
	if err != nil || len(filtered) != 1 || filtered[0].ID != openFlagID {
		t.Fatalf("filtered queue = %#v, err=%v", filtered, err)
	}
	if _, _, err = service.BrandQueue(ctx, brand, fixture.campaignID, QueueQuery{Sort: "content_url"}); err != ErrInvalidQuery {
		t.Fatalf("invalid sort error = %v", err)
	}
	if _, _, err = service.BrandQueue(ctx, otherBrand, fixture.campaignID, QueueQuery{}); err != ErrCampaignNotFound {
		t.Fatalf("unrelated brand queue error = %v", err)
	}
	adminQueue, adminPage, err := service.AdminQueue(ctx, admin, QueueQuery{CampaignID: fixture.campaignID, BrandID: moderationBrandOne, Page: 1, Limit: 2, Sort: "status"})
	if err != nil || len(adminQueue) != 2 || adminPage.TotalItems != 6 {
		t.Fatalf("admin queue = %d, page=%#v, err=%v", len(adminQueue), adminPage, err)
	}
	detail, err := service.BrandDetail(ctx, brand, fixture.campaignID, protectedID)
	if err != nil || len(detail.Requirements) != 1 || detail.ClipperDisplayName == "" {
		t.Fatalf("brand detail = %#v, err=%v", detail, err)
	}
	if _, err = service.BrandDetail(ctx, brand, moderationOtherRoute, protectedID); err != ErrSubmissionNotFound {
		t.Fatalf("detail mismatch error = %v", err)
	}
	if _, err = service.AdminDetail(ctx, admin, protectedID); err != nil {
		t.Fatal(err)
	}

	var actionAudits, resolutionAudits int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_id=$1 AND action='submission.approved'`, approvedID).Scan(&actionAudits); err != nil || actionAudits != 1 {
		t.Fatalf("approval audits = %d, err=%v", actionAudits, err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_id=$1 AND action='submission.flag_resolved'`, resolvedID).Scan(&resolutionAudits); err != nil || resolutionAudits != 1 {
		t.Fatalf("resolution audits = %d, err=%v", resolutionAudits, err)
	}
}

func TestModerationConcurrencyAllowsOneDecision(t *testing.T) {
	service, pool, ctx := moderationIntegration(t)
	fixture := createModerationFixture(t, pool, ctx)
	submissionID := createModerationSubmission(t, pool, ctx, fixture, "TIKTOK")
	brand := moderationActor(moderationBrandOne, auth.RoleBrand, "brand@klipforge.local")
	admin := moderationActor(moderationAdmin, auth.RoleAdmin, "admin@klipforge.local")

	type result struct {
		submission Submission
		err        error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	go func() {
		<-start
		submission, err := service.BrandApprove(ctx, brand, fixture.campaignID, submissionID, ActionInput{RequestID: "concurrent-approve", IPAddress: "127.0.0.1"})
		results <- result{submission, err}
	}()
	go func() {
		<-start
		submission, err := service.AdminReject(ctx, admin, submissionID, ActionInput{Reason: "Concurrent rejection.", RequestID: "concurrent-reject", IPAddress: "127.0.0.1"})
		results <- result{submission, err}
	}()
	close(start)
	first, second := <-results, <-results
	successes := 0
	var winning Submission
	for _, outcome := range []result{first, second} {
		if outcome.err == nil {
			successes++
			winning = outcome.submission
			continue
		}
		domain := AsError(outcome.err)
		if domain == nil || domain.HTTPStatus != 409 {
			t.Fatalf("losing result error = %v", outcome.err)
		}
	}
	if successes != 1 || winning.ReviewedByUserID == nil || winning.ReviewedAt == nil {
		t.Fatalf("successes=%d winning=%#v", successes, winning)
	}
	var auditCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_id=$1 AND action IN ('submission.approved','submission.rejected')`, submissionID).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("concurrent audit count=%d err=%v", auditCount, err)
	}

	secondSubmissionID := createModerationSubmission(t, pool, ctx, fixture, "INSTAGRAM")
	approveResults := make(chan result, 2)
	approveStart := make(chan struct{})
	for range 2 {
		go func() {
			<-approveStart
			submission, err := service.AdminApprove(ctx, admin, secondSubmissionID, ActionInput{RequestID: "double-approve", IPAddress: "127.0.0.1"})
			approveResults <- result{submission, err}
		}()
	}
	close(approveStart)
	approveSuccesses := 0
	for range 2 {
		outcome := <-approveResults
		if outcome.err == nil {
			approveSuccesses++
			continue
		}
		if domain := AsError(outcome.err); domain == nil || domain.HTTPStatus != 409 {
			t.Fatalf("double-approval losing error = %v", outcome.err)
		}
	}
	if approveSuccesses != 1 {
		t.Fatalf("double-approval successes = %d", approveSuccesses)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_id=$1 AND action='submission.approved'`, secondSubmissionID).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("double-approval audit count=%d err=%v", auditCount, err)
	}
}

func TestModerationTransactionRollbackAndSchemaConstraints(t *testing.T) {
	service, pool, ctx := moderationIntegration(t)
	fixture := createModerationFixture(t, pool, ctx)
	brand := moderationActor(moderationBrandOne, auth.RoleBrand, "brand@klipforge.local")
	submissionID := createModerationSubmission(t, pool, ctx, fixture, "TIKTOK")

	if _, err := service.BrandApprove(ctx, brand, fixture.campaignID, submissionID, ActionInput{RequestID: "rollback", IPAddress: "not-an-ip"}); err == nil {
		t.Fatal("expected audit insert failure")
	}
	var status string
	var reviewer *string
	if err := pool.QueryRow(ctx, `SELECT status,reviewed_by_user_id FROM clip_submissions WHERE id=$1`, submissionID).Scan(&status, &reviewer); err != nil || status != StatusPending || reviewer != nil {
		t.Fatalf("rolled back status=%s reviewer=%v err=%v", status, reviewer, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE clip_submissions SET status='APPROVED' WHERE id=$1`, submissionID); err == nil {
		t.Fatal("expected moderation-state database constraint failure")
	}

	invalidSubmissionID := createModerationSubmission(t, pool, ctx, fixture, "INSTAGRAM")
	if _, err := pool.Exec(ctx, `UPDATE clip_submissions SET content_url='https://example.com/not-a-platform-url' WHERE id=$1`, invalidSubmissionID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.BrandApprove(ctx, brand, fixture.campaignID, invalidSubmissionID, ActionInput{RequestID: "invalid-url", IPAddress: "127.0.0.1"}); err != ErrSubmissionNotReady {
		t.Fatalf("invalid stored URL error = %v", err)
	}
}

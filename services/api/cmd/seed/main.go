package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	adminPasswordHash   = "$2a$12$MK/t5MqTGVdxP8k0dLd6Hu.zP9dyAEyfPbyfZHyXu5GUXLAQjYH4y"
	brandPasswordHash   = "$2a$12$OBzBepyQ0E5m3KCZIs0Z7ui2.3.5/v2N3lFwa7PfFGF4/X71fqOUe"
	clipperPasswordHash = "$2a$12$ZDpSqMwUZom2t3.sG1zy4uKYdtnEWQ2QacoIJ074JlNIWTgZznLlm"
)

type userFixture struct {
	id           string
	email        string
	passwordHash string
	role         string
	displayName  string
	bio          string
}

type campaignFixture struct {
	id              string
	brandID         string
	name            string
	description     string
	status          string
	budget          string
	remainingBudget string
	cpm             string
	maxPayout       string
	startDate       string
	endDate         string
}

type participantFixture struct {
	id         string
	campaignID string
	clipperID  string
	status     string
}

type submissionFixture struct {
	id            string
	campaignID    string
	participantID string
	contentURL    string
	platform      string
	status        string
	reviewerID    *string
	reviewNote    *string
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("create seed PostgreSQL pool: %w", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := seed(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}

	var users, campaigns, submissions, payouts int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&users); err != nil {
		return fmt.Errorf("count seeded users: %w", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM campaigns`).Scan(&campaigns); err != nil {
		return fmt.Errorf("count seeded campaigns: %w", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM clip_submissions`).Scan(&submissions); err != nil {
		return fmt.Errorf("count seeded submissions: %w", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM payouts`).Scan(&payouts); err != nil {
		return fmt.Errorf("count seeded payouts: %w", err)
	}

	fmt.Printf("Seed complete: %d users, %d campaigns, %d submissions, %d payouts.\n", users, campaigns, submissions, payouts)
	return nil
}

func seed(ctx context.Context, tx pgx.Tx) error {
	users := []userFixture{
		{"00000000-0000-0000-0000-000000000001", "admin@klipforge.local", adminPasswordHash, "ADMIN", "KlipForge Admin", "Local development administrator."},
		{"00000000-0000-0000-0000-000000000101", "brand@klipforge.local", brandPasswordHash, "BRAND", "Northstar Studio", "Primary local development brand."},
		{"00000000-0000-0000-0000-000000000102", "brand-summit@klipforge.local", brandPasswordHash, "BRAND", "Summit Goods", "Outdoor goods campaign team."},
		{"00000000-0000-0000-0000-000000000103", "brand-pixel@klipforge.local", brandPasswordHash, "BRAND", "Pixel Pantry", "Food and beverage campaign team."},
		{"00000000-0000-0000-0000-000000000201", "clipper@klipforge.local", clipperPasswordHash, "CLIPPER", "Ari Lin", "Short-form creator and local development clipper."},
		{"00000000-0000-0000-0000-000000000202", "clipper-beryl@klipforge.local", clipperPasswordHash, "CLIPPER", "Beryl Moss", "Lifestyle clipper."},
		{"00000000-0000-0000-0000-000000000203", "clipper-cato@klipforge.local", clipperPasswordHash, "CLIPPER", "Cato Reyes", "Technology clipper."},
		{"00000000-0000-0000-0000-000000000204", "clipper-dara@klipforge.local", clipperPasswordHash, "CLIPPER", "Dara Noor", "Comedy and culture clipper."},
		{"00000000-0000-0000-0000-000000000205", "clipper-eli@klipforge.local", clipperPasswordHash, "CLIPPER", "Eli Park", "Wellness clipper."},
		{"00000000-0000-0000-0000-000000000206", "clipper-finn@klipforge.local", clipperPasswordHash, "CLIPPER", "Finn Vale", "Gaming clipper."},
		{"00000000-0000-0000-0000-000000000207", "clipper-gale@klipforge.local", clipperPasswordHash, "CLIPPER", "Gale Kim", "Beauty clipper."},
		{"00000000-0000-0000-0000-000000000208", "clipper-harper@klipforge.local", clipperPasswordHash, "CLIPPER", "Harper Quinn", "Food clipper."},
	}
	for _, user := range users {
		if err := exec(ctx, tx, "seed user", `
			INSERT INTO users (id, email, password_hash, role)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO UPDATE SET
				email = EXCLUDED.email,
				password_hash = EXCLUDED.password_hash,
				role = EXCLUDED.role`, user.id, user.email, user.passwordHash, user.role); err != nil {
			return err
		}
		if err := exec(ctx, tx, "seed profile", `
			INSERT INTO profiles (user_id, display_name, bio)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id) DO UPDATE SET
				display_name = EXCLUDED.display_name,
				bio = EXCLUDED.bio`, user.id, user.displayName, user.bio); err != nil {
			return err
		}
	}

	campaigns := []campaignFixture{
		{"00000000-0000-0000-0000-000000000301", "00000000-0000-0000-0000-000000000101", "Northstar launch clips", "Spring product-launch campaign for concise creator explainers.", "ACTIVE", "12000.00", "8425.00", "18.5000", "300.00", "2026-01-10", "2026-03-31"},
		{"00000000-0000-0000-0000-000000000302", "00000000-0000-0000-0000-000000000101", "Northstar field stories", "Creator stories from the field-testing program.", "PAUSED", "8000.00", "6120.00", "16.2500", "250.00", "2026-02-01", "2026-04-15"},
		{"00000000-0000-0000-0000-000000000303", "00000000-0000-0000-0000-000000000102", "Summit trail kit", "Trail kit awareness and packing-guide submissions.", "DRAFT", "6000.00", "6000.00", "14.0000", "200.00", "2026-04-01", "2026-05-31"},
		{"00000000-0000-0000-0000-000000000304", "00000000-0000-0000-0000-000000000102", "Summit winter recap", "Completed winter campaign recap content.", "COMPLETED", "9500.00", "0.00", "20.0000", "350.00", "2025-10-01", "2025-12-15"},
		{"00000000-0000-0000-0000-000000000305", "00000000-0000-0000-0000-000000000103", "Pixel Pantry pantry reset", "Completed pantry reset recipe and organization clips.", "COMPLETED", "7200.00", "0.00", "15.7500", "220.00", "2025-11-01", "2025-12-20"},
		{"00000000-0000-0000-0000-000000000306", "00000000-0000-0000-0000-000000000103", "Pixel Pantry lunch sprint", "Cancelled lunch recipe sprint retained for audit examples.", "CANCELLED", "4000.00", "4000.00", "13.0000", "180.00", "2026-01-05", "2026-02-20"},
	}
	for _, campaign := range campaigns {
		if err := exec(ctx, tx, "seed campaign", `
			INSERT INTO campaigns (
				id, brand_id, name, description, status, budget_amount, remaining_budget_amount,
				cpm_amount, max_payout_per_clip, currency_code, start_date, end_date
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'USD', $10, $11)
			ON CONFLICT (id) DO UPDATE SET
				brand_id = EXCLUDED.brand_id,
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				status = EXCLUDED.status,
				budget_amount = EXCLUDED.budget_amount,
				remaining_budget_amount = EXCLUDED.remaining_budget_amount,
				cpm_amount = EXCLUDED.cpm_amount,
				max_payout_per_clip = EXCLUDED.max_payout_per_clip,
				start_date = EXCLUDED.start_date,
				end_date = EXCLUDED.end_date`,
			campaign.id, campaign.brandID, campaign.name, campaign.description, campaign.status, campaign.budget,
			campaign.remainingBudget, campaign.cpm, campaign.maxPayout, campaign.startDate, campaign.endDate); err != nil {
			return err
		}
	}

	platforms := []struct{ campaignID, platform string }{
		{"00000000-0000-0000-0000-000000000301", "TIKTOK"}, {"00000000-0000-0000-0000-000000000301", "INSTAGRAM"},
		{"00000000-0000-0000-0000-000000000302", "YOUTUBE"}, {"00000000-0000-0000-0000-000000000302", "TIKTOK"},
		{"00000000-0000-0000-0000-000000000303", "INSTAGRAM"}, {"00000000-0000-0000-0000-000000000304", "YOUTUBE"},
		{"00000000-0000-0000-0000-000000000304", "INSTAGRAM"},
		{"00000000-0000-0000-0000-000000000305", "TIKTOK"}, {"00000000-0000-0000-0000-000000000306", "INSTAGRAM"},
	}
	for _, platform := range platforms {
		if err := exec(ctx, tx, "seed campaign platform", `
			INSERT INTO campaign_platforms (campaign_id, platform) VALUES ($1, $2)
			ON CONFLICT (campaign_id, platform) DO NOTHING`, platform.campaignID, platform.platform); err != nil {
			return err
		}
	}

	requirements := []struct {
		id, campaignID, requirementType, description string
		position                                     int
	}{
		{"00000000-0000-0000-0000-000000000401", "00000000-0000-0000-0000-000000000301", "CONTENT", "Show the product in use within the first five seconds.", 1},
		{"00000000-0000-0000-0000-000000000402", "00000000-0000-0000-0000-000000000301", "DISCLOSURE", "Include the required paid partnership disclosure.", 2},
		{"00000000-0000-0000-0000-000000000403", "00000000-0000-0000-0000-000000000302", "CONTENT", "Explain one field-testing insight in the clip.", 1},
		{"00000000-0000-0000-0000-000000000404", "00000000-0000-0000-0000-000000000304", "MENTION", "Mention the winter trail kit by name.", 1},
		{"00000000-0000-0000-0000-000000000405", "00000000-0000-0000-0000-000000000305", "HASHTAG", "Include the approved pantry reset hashtag.", 1},
	}
	for _, requirement := range requirements {
		if err := exec(ctx, tx, "seed campaign requirement", `
			INSERT INTO campaign_requirements (id, campaign_id, requirement_type, description, position)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (id) DO UPDATE SET
				requirement_type = EXCLUDED.requirement_type,
				description = EXCLUDED.description,
				position = EXCLUDED.position`, requirement.id, requirement.campaignID, requirement.requirementType, requirement.description, requirement.position); err != nil {
			return err
		}
	}

	participants := []participantFixture{
		{"00000000-0000-0000-0000-000000000501", "00000000-0000-0000-0000-000000000301", "00000000-0000-0000-0000-000000000201", "ACCEPTED"},
		{"00000000-0000-0000-0000-000000000502", "00000000-0000-0000-0000-000000000301", "00000000-0000-0000-0000-000000000202", "ACCEPTED"},
		{"00000000-0000-0000-0000-000000000503", "00000000-0000-0000-0000-000000000302", "00000000-0000-0000-0000-000000000203", "ACCEPTED"},
		{"00000000-0000-0000-0000-000000000504", "00000000-0000-0000-0000-000000000302", "00000000-0000-0000-0000-000000000204", "ACCEPTED"},
		{"00000000-0000-0000-0000-000000000505", "00000000-0000-0000-0000-000000000303", "00000000-0000-0000-0000-000000000205", "PENDING"},
		{"00000000-0000-0000-0000-000000000506", "00000000-0000-0000-0000-000000000304", "00000000-0000-0000-0000-000000000206", "ACCEPTED"},
		{"00000000-0000-0000-0000-000000000507", "00000000-0000-0000-0000-000000000304", "00000000-0000-0000-0000-000000000207", "ACCEPTED"},
		{"00000000-0000-0000-0000-000000000508", "00000000-0000-0000-0000-000000000305", "00000000-0000-0000-0000-000000000208", "ACCEPTED"},
		{"00000000-0000-0000-0000-000000000509", "00000000-0000-0000-0000-000000000306", "00000000-0000-0000-0000-000000000201", "PENDING"},
	}
	for _, participant := range participants {
		if err := exec(ctx, tx, "seed campaign participant", `
			INSERT INTO campaign_participants (id, campaign_id, clipper_id, status, joined_at)
			VALUES ($1, $2, $3, $4, CASE WHEN $4 = 'ACCEPTED' THEN '2026-01-12T09:00:00Z'::timestamptz ELSE NULL END)
			ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, joined_at = EXCLUDED.joined_at`,
			participant.id, participant.campaignID, participant.clipperID, participant.status); err != nil {
			return err
		}
	}

	adminID := "00000000-0000-0000-0000-000000000001"
	rejectedNote := "The clip does not yet include the required field-testing insight."
	flaggedNote := "The submission needs a policy review before it can be approved."
	submissions := []submissionFixture{
		{"00000000-0000-0000-0000-000000000601", "00000000-0000-0000-0000-000000000301", "00000000-0000-0000-0000-000000000501", "https://www.tiktok.com/@ari.lin/video/100000000001", "TIKTOK", "APPROVED", &adminID, nil},
		{"00000000-0000-0000-0000-000000000602", "00000000-0000-0000-0000-000000000301", "00000000-0000-0000-0000-000000000502", "https://www.instagram.com/reel/C1000000002", "INSTAGRAM", "PENDING", nil, nil},
		{"00000000-0000-0000-0000-000000000603", "00000000-0000-0000-0000-000000000302", "00000000-0000-0000-0000-000000000503", "https://www.youtube.com/shorts/100000000003", "YOUTUBE", "REJECTED", &adminID, &rejectedNote},
		{"00000000-0000-0000-0000-000000000604", "00000000-0000-0000-0000-000000000302", "00000000-0000-0000-0000-000000000504", "https://www.tiktok.com/@dara.noor/video/100000000004", "TIKTOK", "FLAGGED", &adminID, &flaggedNote},
		{"00000000-0000-0000-0000-000000000605", "00000000-0000-0000-0000-000000000304", "00000000-0000-0000-0000-000000000506", "https://www.instagram.com/reel/C1000000005", "INSTAGRAM", "APPROVED", &adminID, nil},
		{"00000000-0000-0000-0000-000000000606", "00000000-0000-0000-0000-000000000304", "00000000-0000-0000-0000-000000000507", "https://www.youtube.com/shorts/100000000006", "YOUTUBE", "APPROVED", &adminID, nil},
		{"00000000-0000-0000-0000-000000000607", "00000000-0000-0000-0000-000000000305", "00000000-0000-0000-0000-000000000508", "https://www.tiktok.com/@harper.quinn/video/100000000007", "TIKTOK", "APPROVED", &adminID, nil},
	}
	for _, submission := range submissions {
		if err := exec(ctx, tx, "seed clip submission", `
			INSERT INTO clip_submissions (
				id, campaign_id, participant_id, content_url, platform, status, submitted_at,
				reviewed_at, reviewed_by_user_id, review_note
			) VALUES (
				$1, $2, $3, $4, $5, $6, '2026-01-15T10:00:00Z'::timestamptz,
				CASE WHEN $7::uuid IS NULL THEN NULL ELSE '2026-01-16T10:00:00Z'::timestamptz END, $7, $8
			)
			ON CONFLICT (id) DO UPDATE SET
				content_url = EXCLUDED.content_url,
				platform = EXCLUDED.platform,
				status = EXCLUDED.status,
				reviewed_at = EXCLUDED.reviewed_at,
				reviewed_by_user_id = EXCLUDED.reviewed_by_user_id,
				review_note = EXCLUDED.review_note`,
			submission.id, submission.campaignID, submission.participantID, submission.contentURL, submission.platform,
			submission.status, submission.reviewerID, submission.reviewNote); err != nil {
			return err
		}
	}

	metricSnapshots := []struct {
		id, submissionID, capturedAt          string
		views, likes, comments, shares, saves int64
	}{
		{"00000000-0000-0000-0000-000000000701", "00000000-0000-0000-0000-000000000601", "2026-01-20T00:00:00Z", 120000, 9800, 420, 760, 310},
		{"00000000-0000-0000-0000-000000000702", "00000000-0000-0000-0000-000000000601", "2026-01-27T00:00:00Z", 184000, 14300, 620, 1100, 490},
		{"00000000-0000-0000-0000-000000000703", "00000000-0000-0000-0000-000000000605", "2025-11-15T00:00:00Z", 98000, 7600, 330, 510, 280},
		{"00000000-0000-0000-0000-000000000704", "00000000-0000-0000-0000-000000000605", "2025-11-22T00:00:00Z", 156000, 11900, 540, 820, 430},
		{"00000000-0000-0000-0000-000000000705", "00000000-0000-0000-0000-000000000606", "2025-11-22T00:00:00Z", 141000, 10500, 470, 690, 350},
		{"00000000-0000-0000-0000-000000000706", "00000000-0000-0000-0000-000000000607", "2025-12-05T00:00:00Z", 112000, 8700, 390, 610, 290},
	}
	for _, snapshot := range metricSnapshots {
		if err := exec(ctx, tx, "seed metric snapshot", `
			INSERT INTO metric_snapshots (
				id, submission_id, captured_at, views_count, likes_count, comments_count, shares_count, saves_count
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				captured_at = EXCLUDED.captured_at,
				views_count = EXCLUDED.views_count,
				likes_count = EXCLUDED.likes_count,
				comments_count = EXCLUDED.comments_count,
				shares_count = EXCLUDED.shares_count,
				saves_count = EXCLUDED.saves_count`,
			snapshot.id, snapshot.submissionID, snapshot.capturedAt, snapshot.views, snapshot.likes, snapshot.comments, snapshot.shares, snapshot.saves); err != nil {
			return err
		}
	}

	idempotencyKeys := []struct{ id, key string }{
		{"00000000-0000-0000-0000-000000000801", "seed-payout-processed"},
		{"00000000-0000-0000-0000-000000000802", "seed-payout-approved"},
		{"00000000-0000-0000-0000-000000000803", "seed-payout-pending"},
		{"00000000-0000-0000-0000-000000000804", "seed-payout-failed"},
	}
	for _, key := range idempotencyKeys {
		if err := exec(ctx, tx, "seed idempotency key", `
			INSERT INTO idempotency_keys (
				id, actor_user_id, scope, idempotency_key, request_hash, status, response_code, response_body, expires_at
			) VALUES ($1, $2, 'payout.create', $3, 'seed-fixture-request', 'COMPLETED', 201, '{}'::jsonb, '2027-01-01T00:00:00Z')
			ON CONFLICT (id) DO UPDATE SET
				idempotency_key = EXCLUDED.idempotency_key,
				status = EXCLUDED.status,
				response_code = EXCLUDED.response_code,
				response_body = EXCLUDED.response_body,
				expires_at = EXCLUDED.expires_at`, key.id, adminID, key.key); err != nil {
			return err
		}
	}

	payouts := []struct {
		id, campaignID, submissionID, participantID, idempotencyKeyID, status, amount, processorReference string
		approved, processed                                                                               bool
	}{
		{"00000000-0000-0000-0000-000000000901", "00000000-0000-0000-0000-000000000301", "00000000-0000-0000-0000-000000000601", "00000000-0000-0000-0000-000000000501", "00000000-0000-0000-0000-000000000801", "PROCESSED", "250.00", "seed-processor-0001", true, true},
		{"00000000-0000-0000-0000-000000000902", "00000000-0000-0000-0000-000000000304", "00000000-0000-0000-0000-000000000605", "00000000-0000-0000-0000-000000000506", "00000000-0000-0000-0000-000000000802", "APPROVED", "180.00", "seed-processor-0002", true, false},
		{"00000000-0000-0000-0000-000000000903", "00000000-0000-0000-0000-000000000304", "00000000-0000-0000-0000-000000000606", "00000000-0000-0000-0000-000000000507", "00000000-0000-0000-0000-000000000803", "PENDING", "160.00", "", false, false},
		{"00000000-0000-0000-0000-000000000904", "00000000-0000-0000-0000-000000000305", "00000000-0000-0000-0000-000000000607", "00000000-0000-0000-0000-000000000508", "00000000-0000-0000-0000-000000000804", "FAILED", "220.00", "", false, false},
	}
	for _, payout := range payouts {
		if err := exec(ctx, tx, "seed payout", `
			INSERT INTO payouts (
				id, campaign_id, submission_id, participant_id, idempotency_key_id, status, amount,
				currency_code, processor_reference, approved_by_user_id, approved_at, processed_at, failure_reason, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, 'USD', NULLIF($8, ''),
				CASE WHEN $9 THEN $10::uuid ELSE NULL END,
				CASE WHEN $9 THEN '2026-01-18T09:00:00Z'::timestamptz ELSE NULL END,
				CASE WHEN $11 THEN '2026-01-19T09:00:00Z'::timestamptz ELSE NULL END,
				CASE WHEN $6 = 'FAILED' THEN 'Seeded processor failure example.' ELSE NULL END,
				'2025-12-01T09:00:00Z'::timestamptz
			)
			ON CONFLICT (id) DO UPDATE SET
				status = EXCLUDED.status,
				amount = EXCLUDED.amount,
				processor_reference = EXCLUDED.processor_reference,
				approved_by_user_id = EXCLUDED.approved_by_user_id,
				approved_at = EXCLUDED.approved_at,
				processed_at = EXCLUDED.processed_at,
				failure_reason = EXCLUDED.failure_reason`,
			payout.id, payout.campaignID, payout.submissionID, payout.participantID, payout.idempotencyKeyID,
			payout.status, payout.amount, payout.processorReference, payout.approved, adminID, payout.processed); err != nil {
			return err
		}
	}

	notifications := []struct {
		id, userID, notificationType, title, body, data string
		readAt                                          *string
	}{
		{"00000000-0000-0000-0000-000000000a01", "00000000-0000-0000-0000-000000000101", "SUBMISSION_APPROVED", "A submission is ready", "A Northstar launch clip has been approved.", `{"submission_id":"00000000-0000-0000-0000-000000000601"}`, nil},
		{"00000000-0000-0000-0000-000000000a02", "00000000-0000-0000-0000-000000000202", "SUBMISSION_PENDING", "Submission received", "Your Northstar launch clip is awaiting review.", `{"submission_id":"00000000-0000-0000-0000-000000000602"}`, nil},
		{"00000000-0000-0000-0000-000000000a03", "00000000-0000-0000-0000-000000000203", "SUBMISSION_REJECTED", "Revision requested", "Your field story needs one required insight before resubmission.", `{"submission_id":"00000000-0000-0000-0000-000000000603"}`, strPtr("2026-01-16T12:00:00Z")},
		{"00000000-0000-0000-0000-000000000a04", "00000000-0000-0000-0000-000000000206", "PAYOUT_APPROVED", "Payout approved", "Your completed campaign payout is approved for processing.", `{"payout_id":"00000000-0000-0000-0000-000000000902"}`, nil},
		{"00000000-0000-0000-0000-000000000a05", "00000000-0000-0000-0000-000000000207", "PAYOUT_PENDING", "Payout pending", "Your payout record is pending approval.", `{"payout_id":"00000000-0000-0000-0000-000000000903"}`, nil},
	}
	for _, notification := range notifications {
		if err := exec(ctx, tx, "seed notification", `
			INSERT INTO notifications (id, user_id, notification_type, title, body, data, read_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::timestamptz, '2026-01-15T10:00:00Z'::timestamptz)
			ON CONFLICT (id) DO UPDATE SET
				notification_type = EXCLUDED.notification_type,
				title = EXCLUDED.title,
				body = EXCLUDED.body,
				data = EXCLUDED.data,
				read_at = EXCLUDED.read_at`,
			notification.id, notification.userID, notification.notificationType, notification.title, notification.body, notification.data, notification.readAt); err != nil {
			return err
		}
	}

	auditLogs := []struct{ id, actorID, actorEmail, actorRole, action, entityType, entityID, metadata string }{
		{"00000000-0000-0000-0000-000000000b01", adminID, "admin@klipforge.local", "ADMIN", "campaign.created", "campaign", "00000000-0000-0000-0000-000000000301", `{"source":"seed"}`},
		{"00000000-0000-0000-0000-000000000b02", "00000000-0000-0000-0000-000000000101", "brand@klipforge.local", "BRAND", "campaign.activated", "campaign", "00000000-0000-0000-0000-000000000301", `{"source":"seed"}`},
		{"00000000-0000-0000-0000-000000000b03", adminID, "admin@klipforge.local", "ADMIN", "submission.approved", "clip_submission", "00000000-0000-0000-0000-000000000601", `{"source":"seed"}`},
		{"00000000-0000-0000-0000-000000000b04", adminID, "admin@klipforge.local", "ADMIN", "submission.flagged", "clip_submission", "00000000-0000-0000-0000-000000000604", `{"reason":"policy review","source":"seed"}`},
		{"00000000-0000-0000-0000-000000000b05", adminID, "admin@klipforge.local", "ADMIN", "payout.processed", "payout", "00000000-0000-0000-0000-000000000901", `{"source":"seed"}`},
	}
	for _, audit := range auditLogs {
		if err := exec(ctx, tx, "seed audit log", `
			INSERT INTO audit_logs (id, actor_user_id, actor_email, actor_role, action, entity_type, entity_id, metadata)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
			ON CONFLICT (id) DO UPDATE SET
				actor_user_id = EXCLUDED.actor_user_id,
				actor_email = EXCLUDED.actor_email,
				actor_role = EXCLUDED.actor_role,
				action = EXCLUDED.action,
				entity_type = EXCLUDED.entity_type,
				entity_id = EXCLUDED.entity_id,
				metadata = EXCLUDED.metadata`,
			audit.id, audit.actorID, audit.actorEmail, audit.actorRole, audit.action, audit.entityType, audit.entityID, audit.metadata); err != nil {
			return err
		}
	}

	return nil
}

func exec(ctx context.Context, tx pgx.Tx, name, query string, args ...any) error {
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func strPtr(value string) *string {
	return &value
}

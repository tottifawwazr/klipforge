package operations

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klipforge/klipforge/services/api/internal/auth"
	"os"
	"sync"
	"testing"
	"time"
)

func TestIntegrationPayoutConcurrencyBudgetAndAudit(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	service := NewService(pool)
	actor := auth.Principal{UserID: "00000000-0000-0000-0000-000000000001", Email: "admin@klipforge.local", Role: auth.RoleAdmin, AccountActive: true, SessionActive: true}
	var campaign, participant string
	err = pool.QueryRow(ctx, `INSERT INTO campaigns(brand_id,name,slug,description,brief,status,budget_amount,remaining_budget_amount,cpm_amount,max_payout_per_clip,start_date,end_date)VALUES('00000000-0000-0000-0000-000000000101','Payout concurrency test',$1,'Test campaign description','An integration brief','COMPLETED',15,15,10,10,CURRENT_DATE-2,CURRENT_DATE-1)RETURNING id`, fmt.Sprintf("payout-test-%d", time.Now().UnixNano())).Scan(&campaign)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for _, query := range []string{`DELETE FROM notifications WHERE data->>'campaign_id'=$1`, `DELETE FROM audit_logs WHERE metadata->>'campaign_id'=$1`, `DELETE FROM payouts WHERE campaign_id=$1`, `DELETE FROM idempotency_keys WHERE response_body->>'campaign_id'=$1`, `DELETE FROM clip_submissions WHERE campaign_id=$1`, `DELETE FROM campaigns WHERE id=$1`} {
			_, _ = pool.Exec(context.Background(), query, campaign)
		}
	}()
	if _, err = pool.Exec(ctx, `INSERT INTO campaign_platforms(campaign_id,platform)VALUES($1,'TIKTOK')`, campaign); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO campaign_participants(campaign_id,clipper_id,status)VALUES($1,'00000000-0000-0000-0000-000000000201','ACCEPTED')RETURNING id`, campaign).Scan(&participant); err != nil {
		t.Fatal(err)
	}
	submissions := []string{}
	for i := 0; i < 2; i++ {
		var id string
		err = pool.QueryRow(ctx, `INSERT INTO clip_submissions(campaign_id,participant_id,content_url,platform,status,submitted_at,reviewed_at,reviewed_by_user_id)VALUES($1,$2,$3,'TIKTOK','APPROVED',NOW()-INTERVAL '2 hours',NOW(),'00000000-0000-0000-0000-000000000101')RETURNING id`, campaign, participant, fmt.Sprintf("https://tiktok.com/@payout/video/%s-%d", campaign, i)).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		submissions = append(submissions, id)
		if _, err = pool.Exec(ctx, `INSERT INTO metric_snapshots(submission_id,captured_at,views_count)VALUES($1,NOW(),1000)`, id); err != nil {
			t.Fatal(err)
		}
	}
	quote, err := service.Quote(ctx, actor, submissions[0])
	if err != nil || quote.Amount != "10.00" {
		t.Fatalf("quote %+v %v", quote, err)
	}
	if _, _, err := service.Create(ctx, actor, submissions[0], "stale-quote-"+campaign, "1.00"); !errors.Is(err, ErrQuote) {
		t.Fatalf("changed quote accepted: %v", err)
	}
	var wg sync.WaitGroup
	results := make(chan Payout, 5)
	failures := make(chan error, 5)
	key := "concurrent-" + campaign
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, _, err := service.Create(ctx, actor, submissions[0], key)
			results <- p
			failures <- err
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	id := ""
	for p := range results {
		if id == "" {
			id = p.ID
		}
		if p.ID != id || p.Amount != "10.00" || p.BasisViews != 1000 {
			t.Fatalf("inconsistent payout: %+v", p)
		}
	}
	var remaining string
	var count int
	checkBudget := func(expected string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT remaining_budget_amount::text FROM campaigns WHERE id=$1`, campaign).Scan(&remaining); err != nil || remaining != expected {
			t.Fatalf("budget %s, want %s: %v", remaining, expected, err)
		}
	}
	checkBudget("5.00")
	if err := pool.QueryRow(ctx, `SELECT count(*)FROM payouts WHERE campaign_id=$1`, campaign).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate payout %d %v", count, err)
	}
	if _, _, err := service.Create(ctx, actor, submissions[1], key); !errors.Is(err, ErrKey) {
		t.Fatalf("key reuse allowed: %v", err)
	}
	if _, _, err := service.Create(ctx, actor, submissions[1], "second-"+campaign); !errors.Is(err, ErrBudget) {
		t.Fatalf("budget overspend allowed: %v", err)
	}
	checkBudget("5.00")
	if _, err := service.Transition(ctx, actor, id, "fail", "", "Test processor failure"); err != nil {
		t.Fatal(err)
	}
	checkBudget("15.00")
	if _, err := service.Transition(ctx, actor, id, "fail", "", "Repeated failure"); !errors.Is(err, ErrState) {
		t.Fatal("repeat failure refunded twice")
	}
	checkBudget("15.00")
	if _, err := service.Transition(ctx, actor, id, "retry", "", ""); err != nil {
		t.Fatal(err)
	}
	checkBudget("5.00")
	if _, err := service.Transition(ctx, actor, id, "approve", "", ""); err != nil {
		t.Fatal(err)
	}
	paid, err := service.Transition(ctx, actor, id, "process", "receipt-"+campaign, "")
	if err != nil || paid.Status != "PROCESSED" || paid.Reference == nil {
		t.Fatalf("manual payment record %+v %v", paid, err)
	}
	if _, err := service.Transition(ctx, actor, id, "fail", "", "Cannot undo external payment"); !errors.Is(err, ErrState) {
		t.Fatal("processed payout refunded")
	}
	checkBudget("5.00")
	user := auth.Principal{UserID: "00000000-0000-0000-0000-000000000201", Role: auth.RoleClipper, AccountActive: true, SessionActive: true}
	notes, err := service.Notifications(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	noteID := ""
	for _, n := range notes {
		if n.Title == "Payout status: PROCESSED" {
			noteID = n.ID
			break
		}
	}
	if noteID == "" {
		t.Fatal("payout notification missing")
	}
	other := user
	other.UserID = "00000000-0000-0000-0000-000000000202"
	if err := service.ReadNotification(ctx, other, noteID); !errors.Is(err, ErrNotFound) {
		t.Fatal("other user read notification")
	}
	if err := service.ReadNotification(ctx, user, noteID); err != nil {
		t.Fatal(err)
	}
	visible, err := service.List(ctx, other, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range visible {
		if p.ID == id {
			t.Fatal("another clipper saw private payout")
		}
	}
	if _, err := service.Audit(ctx, user); !errors.Is(err, ErrForbidden) {
		t.Fatal("clipper saw administrative audit")
	}
	var events int
	if err := pool.QueryRow(ctx, `SELECT count(*)FROM audit_logs WHERE entity_id=$1 AND action LIKE 'payout.%'`, id).Scan(&events); err != nil || events != 5 {
		t.Fatalf("audit event count %d %v", events, err)
	}
}

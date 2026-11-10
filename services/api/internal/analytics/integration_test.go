package analytics

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/redis/go-redis/v9"
	"os"
	"testing"
	"time"
)

func TestIntegrationCollectionAuthorizationAndIdempotency(t *testing.T) {
	db, redisURL := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_REDIS_URL")
	if db == "" || redisURL == "" {
		t.Skip("TEST_DATABASE_URL and TEST_REDIS_URL required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatal(err)
	}
	cache := redis.NewClient(options)
	defer cache.Close()
	var campaign, participant, approved, pending string
	err = pool.QueryRow(ctx, `INSERT INTO campaigns(brand_id,name,slug,description,brief,status,budget_amount,remaining_budget_amount,cpm_amount,max_payout_per_clip,start_date,end_date)VALUES('00000000-0000-0000-0000-000000000101','Analytics integration',$1,'Analytics test description','Integration testing brief','ACTIVE',1000,1000,10,100,CURRENT_DATE-1,CURRENT_DATE+1)RETURNING id`, fmt.Sprintf("analytics-%d", time.Now().UnixNano())).Scan(&campaign)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM metric_snapshots WHERE submission_id IN(SELECT id FROM clip_submissions WHERE campaign_id=$1)`, campaign)
		_, _ = pool.Exec(context.Background(), `DELETE FROM clip_submissions WHERE campaign_id=$1`, campaign)
		_, _ = pool.Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1`, campaign)
	}()
	if _, err = pool.Exec(ctx, `INSERT INTO campaign_platforms(campaign_id,platform)VALUES($1,'TIKTOK')`, campaign); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO campaign_participants(campaign_id,clipper_id,status)VALUES($1,'00000000-0000-0000-0000-000000000201','ACCEPTED')RETURNING id`, campaign).Scan(&participant); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO clip_submissions(campaign_id,participant_id,content_url,platform,status,submitted_at,reviewed_at,reviewed_by_user_id)VALUES($1,$2,$3,'TIKTOK','APPROVED',NOW()-INTERVAL '2 hours',NOW(),'00000000-0000-0000-0000-000000000101')RETURNING id`, campaign, participant, "https://tiktok.com/@analytics/video/"+campaign).Scan(&approved); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO clip_submissions(campaign_id,participant_id,content_url,platform,status)VALUES($1,$2,$3,'TIKTOK','PENDING')RETURNING id`, campaign, participant, "https://tiktok.com/@analytics/video/pending-"+campaign).Scan(&pending); err != nil {
		t.Fatal(err)
	}
	collector := NewCollector(pool, cache)
	first, err := collector.Collect(ctx)
	if err != nil || first.Collected < 1 {
		t.Fatalf("collection %+v: %v", first, err)
	}
	repeat, err := collector.Collect(ctx)
	if err != nil || repeat.Collected != 0 {
		t.Fatalf("duplicate collection %+v %v", repeat, err)
	}
	var pendingSnapshots int
	if err := pool.QueryRow(ctx, `SELECT count(*)FROM metric_snapshots WHERE submission_id=$1`, pending).Scan(&pendingSnapshots); err != nil || pendingSnapshots != 0 {
		t.Fatalf("pending submission collected: %d %v", pendingSnapshots, err)
	}
	actor := auth.Principal{UserID: "00000000-0000-0000-0000-000000000101", Role: auth.RoleBrand, AccountActive: true, SessionActive: true}
	service := NewService(pool, cache)
	overview, err := service.Overview(ctx, actor, campaign)
	if err != nil || overview.Approved != 1 || overview.Submissions != 2 || overview.Views <= 0 || overview.Mode != "SIMULATED" {
		t.Fatalf("overview %+v %v", overview, err)
	}
	actor.UserID = "00000000-0000-0000-0000-000000000102"
	if _, err := service.Overview(ctx, actor, campaign); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other brand obtained cached analytics: %v", err)
	}
	actor.UserID = "00000000-0000-0000-0000-000000000201"
	actor.Role = auth.RoleClipper
	ranking, err := service.Leaderboard(ctx, actor, campaign)
	if err != nil || len(ranking) != 1 || ranking[0].Views <= 0 {
		t.Fatalf("leaderboard %+v %v", ranking, err)
	}
	actor.UserID = "00000000-0000-0000-0000-000000000202"
	if _, err := service.Leaderboard(ctx, actor, campaign); !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-participant obtained ranking: %v", err)
	}
	if err := cache.Set(ctx, "metrics:collector:lock", "another-worker", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	defer cache.Del(ctx, "metrics:collector:lock")
	busy, err := collector.Collect(ctx)
	if err != nil || !busy.Busy || busy.Collected != 0 {
		t.Fatalf("coordination %+v %v", busy, err)
	}
	token, _ := cache.Get(ctx, "metrics:collector:lock").Result()
	if token != "another-worker" {
		t.Fatal("collector removed another worker lock")
	}
}

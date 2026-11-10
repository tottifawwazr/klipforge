package youtube

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/redis/go-redis/v9"
)

func TestIntegrationReadOnlyPublicMetrics(t *testing.T) {
	db, redisURL := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_REDIS_URL")
	if db == "" || redisURL == "" {
		t.Skip("isolated PostgreSQL and Redis test URLs required")
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
	err = pool.QueryRow(ctx, `INSERT INTO campaigns(brand_id,name,slug,description,brief,status,budget_amount,remaining_budget_amount,cpm_amount,max_payout_per_clip,start_date,end_date)VALUES('00000000-0000-0000-0000-000000000101','YouTube integration',$1,'YouTube test campaign','Read-only integration testing brief','ACTIVE',1000,1000,10,100,CURRENT_DATE-1,CURRENT_DATE+1)RETURNING id`, fmt.Sprintf("youtube-%d", time.Now().UnixNano())).Scan(&campaign)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM clip_submissions WHERE campaign_id=$1`, campaign)
		_, _ = pool.Exec(context.Background(), `DELETE FROM campaigns WHERE id=$1`, campaign)
	}()
	if err = pool.QueryRow(ctx, `INSERT INTO campaign_participants(campaign_id,clipper_id,status)VALUES($1,'00000000-0000-0000-0000-000000000201','ACCEPTED')RETURNING id`, campaign).Scan(&participant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO campaign_platforms(campaign_id,platform)VALUES($1,'YOUTUBE')`, campaign); err != nil {
		t.Fatal(err)
	}
	videoID := fmt.Sprintf("yt%09d", time.Now().UnixNano()%1_000_000_000)
	if err = pool.QueryRow(ctx, `INSERT INTO clip_submissions(campaign_id,participant_id,content_url,platform,status,submitted_at,reviewed_at,reviewed_by_user_id)VALUES($1,$2,$3,'YOUTUBE','APPROVED',NOW()-INTERVAL '2 hours',NOW(),'00000000-0000-0000-0000-000000000101')RETURNING id`, campaign, participant, "https://www.youtube.com/watch?v="+videoID).Scan(&approved); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `INSERT INTO clip_submissions(campaign_id,participant_id,content_url,platform,status)VALUES($1,$2,$3,'YOUTUBE','PENDING')RETURNING id`, campaign, participant, "https://www.youtube.com/watch?v=abcdefghijk").Scan(&pending); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprintf(w, `{"items":[{"id":%q,"statistics":{"viewCount":"1000","likeCount":"10","commentCount":"2"}}]}`, videoID)
	}))
	defer server.Close()
	service := NewService(pool, cache, "fixture-secret")
	service.provider.endpoint = server.URL
	key := "youtube:public:video:" + videoID
	guard := "youtube:public:request:" + videoID
	budget := "youtube:public:daily:" + time.Now().UTC().Format("2006-01-02")
	defer cache.Del(context.Background(), key, guard, budget)
	brand := auth.Principal{UserID: "00000000-0000-0000-0000-000000000101", Role: auth.RoleBrand, AccountActive: true, SessionActive: true}
	index, err := service.Index(ctx, brand, campaign)
	if err != nil || len(index.Videos) != 1 || index.Videos[0].ID != approved || !index.Configured || index.UsedForPayouts {
		t.Fatalf("bad index: %+v %v", index, err)
	}
	metrics, err := service.Get(ctx, brand, approved, false)
	if err != nil || metrics.Views != 1000 || metrics.UsedForPayouts {
		t.Fatalf("load: %+v %v", metrics, err)
	}
	if _, err = service.Get(ctx, brand, approved, false); err != nil || calls.Load() != 1 {
		t.Fatalf("cache missed: %v calls=%d", err, calls.Load())
	}
	other := brand
	other.UserID = "00000000-0000-0000-0000-000000000102"
	if _, err = service.Get(ctx, other, approved, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-brand cached access: %v", err)
	}
	if _, err = service.Get(ctx, brand, approved, true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("brand forced refresh: %v", err)
	}
	clipper := brand
	clipper.Role = auth.RoleClipper
	clipper.UserID = "00000000-0000-0000-0000-000000000201"
	if _, err = service.Get(ctx, clipper, approved, false); err != nil {
		t.Fatalf("owner denied: %v", err)
	}
	outsider := clipper
	outsider.UserID = "00000000-0000-0000-0000-000000000202"
	if _, err = service.Get(ctx, outsider, approved, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other creator access: %v", err)
	}
	if _, err = service.Get(ctx, brand, pending, false); !errors.Is(err, ErrNotApproved) {
		t.Fatalf("pending clip fetched: %v", err)
	}
	admin := brand
	admin.Role = auth.RoleAdmin
	admin.UserID = "00000000-0000-0000-0000-000000000001"
	if _, err = service.Get(ctx, admin, approved, true); !errors.Is(err, ErrBusy) {
		t.Fatalf("refresh guard bypassed: %v", err)
	}
	disabled := NewService(pool, cache, "")
	if _, err = disabled.Get(ctx, brand, approved, false); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("disabled provider used a cache: %v", err)
	}
	if ttl := cache.TTL(ctx, key).Val(); ttl <= 0 || ttl > 15*time.Minute {
		t.Fatalf("bad retention %s", ttl)
	}
	var snapshots, payouts int
	_ = pool.QueryRow(ctx, `SELECT count(*)FROM metric_snapshots WHERE submission_id=$1`, approved).Scan(&snapshots)
	_ = pool.QueryRow(ctx, `SELECT count(*)FROM payouts WHERE submission_id=$1`, approved).Scan(&payouts)
	if snapshots != 0 || payouts != 0 {
		t.Fatalf("public counters entered simulated ledger: %d %d", snapshots, payouts)
	}
	if err = cache.Del(ctx, key, guard).Err(); err != nil {
		t.Fatal(err)
	}
	if err = cache.Set(ctx, budget, dailyBudget, 48*time.Hour).Err(); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Get(ctx, admin, approved, true); !errors.Is(err, ErrQuota) || calls.Load() != 1 {
		t.Fatalf("daily budget bypassed: %v calls=%d", err, calls.Load())
	}
}

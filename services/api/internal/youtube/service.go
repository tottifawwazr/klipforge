package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/redis/go-redis/v9"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

const dailyBudget = 1000

// The cache is deliberately separate from metric_snapshots and the payout ledger.
type Service struct {
	pool     *pgxpool.Pool
	cache    *redis.Client
	provider *Provider
}

func NewService(pool *pgxpool.Pool, cache *redis.Client, key string) *Service {
	return &Service{pool: pool, cache: cache, provider: NewProvider(key)}
}

type Video struct {
	ID            string `json:"id"`
	CampaignTitle string `json:"campaign_title"`
	URL           string `json:"content_url"`
}
type Index struct {
	Configured        bool    `json:"configured"`
	Source            string  `json:"source"`
	CacheSeconds      int     `json:"cache_seconds"`
	DailyRequestLimit int     `json:"daily_request_limit"`
	UsedForPayouts    bool    `json:"used_for_payouts"`
	Videos            []Video `json:"videos"`
}

func active(actor auth.Principal) bool {
	return actor.UserID != "" && actor.AccountActive && actor.SessionActive && (actor.Role == auth.RoleAdmin || actor.Role == auth.RoleBrand || actor.Role == auth.RoleClipper)
}

const actorFilter = `($1='ADMIN' OR ($1='BRAND' AND c.brand_id::text=$2) OR ($1='CLIPPER' AND cp.clipper_id::text=$2))`

func (s *Service) Index(ctx context.Context, actor auth.Principal, campaign string) (Index, error) {
	if !active(actor) {
		return Index{}, ErrForbidden
	}
	if campaign != "" && !uuidPattern.MatchString(campaign) {
		return Index{}, ErrInvalid
	}
	result := Index{Configured: s.provider.Configured(), Source: "YOUTUBE_DATA_API", CacheSeconds: int(cacheTTL / time.Second), DailyRequestLimit: dailyBudget, Videos: []Video{}}
	rows, err := s.pool.Query(ctx, `SELECT cs.id,c.name,cs.content_url FROM clip_submissions cs JOIN campaigns c ON c.id=cs.campaign_id JOIN campaign_participants cp ON cp.id=cs.participant_id WHERE `+actorFilter+` AND cs.platform='YOUTUBE' AND cs.status='APPROVED' AND ($3='' OR cs.campaign_id::text=$3) ORDER BY cs.submitted_at DESC,cs.id DESC LIMIT 20`, actor.Role, actor.UserID, campaign)
	if err != nil {
		return Index{}, ErrUnavailable
	}
	defer rows.Close()
	for rows.Next() {
		var item Video
		if rows.Scan(&item.ID, &item.CampaignTitle, &item.URL) != nil {
			return Index{}, ErrUnavailable
		}
		result.Videos = append(result.Videos, item)
	}
	if rows.Err() != nil {
		return Index{}, ErrUnavailable
	}
	return result, nil
}
func (s *Service) Get(ctx context.Context, actor auth.Principal, submission string, force bool) (Snapshot, error) {
	if !active(actor) || force && actor.Role != auth.RoleAdmin {
		return Snapshot{}, ErrForbidden
	}
	if !uuidPattern.MatchString(submission) {
		return Snapshot{}, ErrNotFound
	}
	var contentURL, platform, status string
	err := s.pool.QueryRow(ctx, `SELECT cs.content_url,cs.platform,cs.status FROM clip_submissions cs JOIN campaigns c ON c.id=cs.campaign_id JOIN campaign_participants cp ON cp.id=cs.participant_id WHERE `+actorFilter+` AND cs.id=$3`, actor.Role, actor.UserID, submission).Scan(&contentURL, &platform, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return Snapshot{}, ErrNotFound
	}
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	if platform != "YOUTUBE" {
		return Snapshot{}, ErrInvalid
	}
	if status != "APPROVED" {
		return Snapshot{}, ErrNotApproved
	}
	id, err := VideoID(contentURL)
	if err != nil {
		return Snapshot{}, err
	}
	if !s.provider.Configured() {
		return Snapshot{}, ErrNotConfigured
	}
	if s.cache == nil {
		return Snapshot{}, ErrUnavailable
	}
	key := "youtube:public:video:" + id
	if !force {
		raw, err := s.cache.Get(ctx, key).Bytes()
		if err == nil {
			var snapshot Snapshot
			if json.Unmarshal(raw, &snapshot) == nil && snapshot.VideoID == id && snapshot.Source == "YOUTUBE_DATA_API" && snapshot.ExpiresAt.After(time.Now()) {
				return snapshot, nil
			}
		} else if !errors.Is(err, redis.Nil) {
			return Snapshot{}, ErrUnavailable
		}
	}
	// Keep the request guard for a minute even on provider errors; this is a backoff, not a long-lived lock.
	allowed, err := s.cache.SetNX(ctx, "youtube:public:request:"+id, "requested", time.Minute).Result()
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	if !allowed {
		return Snapshot{}, ErrBusy
	}
	budgetKey := "youtube:public:daily:" + time.Now().UTC().Format("2006-01-02")
	used, err := s.cache.Incr(ctx, budgetKey).Result()
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	if used == 1 {
		if s.cache.Expire(ctx, budgetKey, 48*time.Hour).Err() != nil {
			return Snapshot{}, ErrUnavailable
		}
	}
	if used > dailyBudget {
		return Snapshot{}, ErrQuota
	}
	snapshot, err := s.provider.Fetch(ctx, id)
	if err != nil {
		return Snapshot{}, err
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	if s.cache.Set(ctx, key, raw, cacheTTL).Err() != nil {
		return Snapshot{}, ErrUnavailable
	}
	return snapshot, nil
}
func (s *Service) String() string { return fmt.Sprint("read-only public YouTube counters") }

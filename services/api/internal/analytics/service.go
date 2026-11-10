package analytics

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("analytics resource not found")
var ErrForbidden = errors.New("analytics access forbidden")

type Service struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

func NewService(pool *pgxpool.Pool, client *redis.Client) *Service {
	return &Service{pool: pool, redis: client}
}

type Point struct {
	Date        string `json:"date"`
	Views       int64  `json:"views"`
	Engagements int64  `json:"engagements"`
}
type Overview struct {
	Mode           string `json:"metrics_mode"`
	Campaigns      int64  `json:"campaigns"`
	Submissions    int64  `json:"submissions"`
	Approved       int64  `json:"approved_submissions"`
	Metrics        `json:"metrics"`
	EngagementRate float64    `json:"engagement_rate"`
	LastUpdated    *time.Time `json:"last_updated"`
	Timeline       []Point    `json:"timeline"`
}
type Rank struct {
	Rank          int    `json:"rank"`
	ParticipantID string `json:"participant_id"`
	DisplayName   string `json:"display_name"`
	Views         int64  `json:"views"`
	Clips         int64  `json:"approved_clips"`
}

func validID(id string) bool {
	raw := strings.ReplaceAll(id, "-", "")
	_, err := hex.DecodeString(raw)
	return len(id) == 36 && len(raw) == 32 && err == nil && id[8] == '-' && id[13] == '-' && id[18] == '-' && id[23] == '-'
}
func active(actor auth.Principal) bool {
	return actor.UserID != "" && actor.AccountActive && actor.SessionActive && (actor.Role == auth.RoleBrand || actor.Role == auth.RoleClipper || actor.Role == auth.RoleAdmin)
}

func (s *Service) authorize(ctx context.Context, actor auth.Principal, campaignID string) error {
	if !active(actor) {
		return ErrForbidden
	}
	if campaignID == "" {
		return nil
	}
	if !validID(campaignID) {
		return ErrNotFound
	}
	var allowed bool
	err := s.pool.QueryRow(ctx, `SELECT ($2='ADMIN' OR ($2='BRAND' AND c.brand_id::text=$3) OR ($2='CLIPPER' AND EXISTS(SELECT 1 FROM campaign_participants cp WHERE cp.campaign_id=c.id AND cp.clipper_id::text=$3 AND cp.status='ACCEPTED'))) FROM campaigns c WHERE c.id=$1`, campaignID, actor.Role, actor.UserID).Scan(&allowed)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && !allowed {
		return ErrNotFound
	}
	return err
}

func (s *Service) Overview(ctx context.Context, actor auth.Principal, campaignID string) (Overview, error) {
	if err := s.authorize(ctx, actor, campaignID); err != nil {
		return Overview{}, err
	}
	// Authorization precedes cache reads. Actor-scoped keys never share private data.
	key := "analytics:overview:" + actor.Role + ":" + actor.UserID + ":" + campaignID + ":" + s.generation(ctx)
	var result Overview
	if s.redis != nil {
		if raw, err := s.redis.Get(ctx, key).Bytes(); err == nil && json.Unmarshal(raw, &result) == nil {
			return result, nil
		}
	}
	result.Mode = "SIMULATED"
	result.Timeline = []Point{}
	filter := `($1='ADMIN' OR ($1='BRAND' AND c.brand_id::text=$2) OR ($1='CLIPPER' AND cp.clipper_id::text=$2)) AND ($3='' OR c.id::text=$3)`
	err := s.pool.QueryRow(ctx, `SELECT count(DISTINCT c.id),count(DISTINCT cs.id),count(DISTINCT cs.id) FILTER(WHERE cs.status='APPROVED'),COALESCE(sum(m.views_count) FILTER(WHERE cs.status='APPROVED'),0),COALESCE(sum(m.likes_count) FILTER(WHERE cs.status='APPROVED'),0),COALESCE(sum(m.comments_count) FILTER(WHERE cs.status='APPROVED'),0),COALESCE(sum(m.shares_count) FILTER(WHERE cs.status='APPROVED'),0),COALESCE(sum(m.saves_count) FILTER(WHERE cs.status='APPROVED'),0),max(m.captured_at) FILTER(WHERE cs.status='APPROVED') FROM campaigns c LEFT JOIN clip_submissions cs ON cs.campaign_id=c.id LEFT JOIN campaign_participants cp ON cp.id=cs.participant_id LEFT JOIN LATERAL(SELECT * FROM metric_snapshots WHERE submission_id=cs.id ORDER BY captured_at DESC LIMIT 1)m ON true WHERE `+filter, actor.Role, actor.UserID, campaignID).Scan(&result.Campaigns, &result.Submissions, &result.Approved, &result.Views, &result.Likes, &result.Comments, &result.Shares, &result.Saves, &result.LastUpdated)
	if err != nil {
		return Overview{}, err
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM campaigns c WHERE ($1='ADMIN' OR ($1='BRAND' AND c.brand_id::text=$2) OR ($1='CLIPPER' AND EXISTS(SELECT 1 FROM campaign_participants cp WHERE cp.campaign_id=c.id AND cp.clipper_id::text=$2 AND cp.status='ACCEPTED'))) AND ($3='' OR c.id::text=$3)`, actor.Role, actor.UserID, campaignID).Scan(&result.Campaigns); err != nil {
		return Overview{}, err
	}
	if result.Views > 0 {
		result.EngagementRate = float64(result.Likes+result.Comments+result.Shares+result.Saves) / float64(result.Views) * 100
	}
	rows, err := s.pool.Query(ctx, `SELECT snapshot_day::text,COALESCE(sum(views_count),0),COALESCE(sum(likes_count+comments_count+shares_count+saves_count),0) FROM(SELECT DISTINCT ON (m.submission_id,(m.captured_at AT TIME ZONE 'UTC')::date) (m.captured_at AT TIME ZONE 'UTC')::date AS snapshot_day,m.* FROM metric_snapshots m JOIN clip_submissions cs ON cs.id=m.submission_id JOIN campaigns c ON c.id=cs.campaign_id JOIN campaign_participants cp ON cp.id=cs.participant_id WHERE cs.status='APPROVED' AND m.captured_at>=NOW()-INTERVAL '30 days' AND `+filter+` ORDER BY m.submission_id,(m.captured_at AT TIME ZONE 'UTC')::date,m.captured_at DESC) daily GROUP BY snapshot_day ORDER BY snapshot_day`, actor.Role, actor.UserID, campaignID)
	if err != nil {
		return Overview{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var point Point
		if err := rows.Scan(&point.Date, &point.Views, &point.Engagements); err != nil {
			return Overview{}, err
		}
		result.Timeline = append(result.Timeline, point)
	}
	if err := rows.Err(); err != nil {
		return Overview{}, err
	}
	if s.redis != nil {
		if raw, err := json.Marshal(result); err == nil {
			_ = s.redis.Set(ctx, key, raw, 20*time.Second).Err()
		}
	}
	return result, nil
}

func (s *Service) generation(ctx context.Context) string {
	if s.redis != nil {
		if value, err := s.redis.Get(ctx, "analytics:generation").Result(); err == nil {
			return value
		}
	}
	return "0"
}
func (s *Service) Leaderboard(ctx context.Context, actor auth.Principal, campaignID string) ([]Rank, error) {
	if campaignID == "" {
		return nil, ErrNotFound
	}
	if err := s.authorize(ctx, actor, campaignID); err != nil {
		return nil, err
	}
	key := "analytics:leaderboard:" + campaignID + ":" + s.generation(ctx)
	if s.redis != nil {
		if raw, err := s.redis.Get(ctx, key+":rows").Bytes(); err == nil {
			var cached []Rank
			ids, rankErr := s.redis.ZRevRange(ctx, key, 0, 99).Result()
			if json.Unmarshal(raw, &cached) == nil && rankErr == nil && len(ids) == len(cached) {
				byID := map[string]Rank{}
				for _, item := range cached {
					byID[item.ParticipantID] = item
				}
				result := []Rank{}
				for _, id := range ids {
					item, ok := byID[id]
					if !ok {
						break
					}
					item.Rank = len(result) + 1
					result = append(result, item)
				}
				if len(result) == len(cached) {
					return result, nil
				}
			}
		}
	}
	rows, err := s.pool.Query(ctx, `SELECT cp.id,p.display_name,COALESCE(sum(m.views_count),0),count(cs.id) FROM campaign_participants cp JOIN profiles p ON p.user_id=cp.clipper_id JOIN clip_submissions cs ON cs.participant_id=cp.id AND cs.status='APPROVED' LEFT JOIN LATERAL(SELECT views_count FROM metric_snapshots WHERE submission_id=cs.id ORDER BY captured_at DESC LIMIT 1)m ON true WHERE cp.campaign_id=$1 AND cp.status='ACCEPTED' GROUP BY cp.id,p.display_name ORDER BY sum(COALESCE(m.views_count,0)) DESC,cp.id DESC LIMIT 100`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Rank{}
	for rows.Next() {
		var item Rank
		if err := rows.Scan(&item.ParticipantID, &item.DisplayName, &item.Views, &item.Clips); err != nil {
			return nil, err
		}
		item.Rank = len(items) + 1
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if s.redis != nil {
		raw, err := json.Marshal(items)
		if err == nil {
			_, _ = s.redis.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Del(ctx, key)
				for _, item := range items {
					pipe.ZAdd(ctx, key, redis.Z{Score: float64(item.Views), Member: item.ParticipantID})
				}
				pipe.Expire(ctx, key, 2*time.Minute)
				pipe.Set(ctx, key+":rows", raw, 2*time.Minute)
				return nil
			})
		}
	}
	return items, nil
}

func ErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrNotFound):
		return "Analytics resource not found."
	case errors.Is(err, ErrForbidden):
		return "You do not have access to these analytics."
	default:
		return fmt.Sprint("Analytics are temporarily unavailable.")
	}
}

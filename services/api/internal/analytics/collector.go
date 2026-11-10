package analytics

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Collector struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

func NewCollector(pool *pgxpool.Pool, client *redis.Client) *Collector {
	return &Collector{pool: pool, redis: client}
}

type Collection struct {
	Mode       string    `json:"metrics_mode"`
	Collected  int64     `json:"collected"`
	Busy       bool      `json:"busy"`
	CapturedAt time.Time `json:"captured_at"`
}

func (c *Collector) Collect(parent context.Context) (Collection, error) {
	ctx, cancel := context.WithTimeout(parent, 40*time.Second)
	defer cancel()
	now := time.Now().UTC()
	bucket := now.Truncate(time.Minute)
	result := Collection{Mode: "SIMULATED", CapturedAt: bucket}
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return result, err
	}
	token := hex.EncodeToString(tokenBytes)
	locked, err := c.redis.SetNX(ctx, "metrics:collector:lock", token, time.Minute).Result()
	if err != nil {
		return result, err
	}
	if !locked {
		result.Busy = true
		return result, nil
	}
	defer func() {
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer releaseCancel()
		_ = c.redis.Eval(releaseCtx, `if redis.call('GET',KEYS[1])==ARGV[1] then return redis.call('DEL',KEYS[1]) end return 0`, []string{"metrics:collector:lock"}, token).Err()
	}()
	rows, err := c.pool.Query(ctx, `SELECT cs.id,cs.platform,cs.submitted_at,cs.campaign_id FROM clip_submissions cs JOIN campaigns c ON c.id=cs.campaign_id WHERE cs.status='APPROVED' AND c.status IN('ACTIVE','PAUSED','COMPLETED') AND cs.submitted_at<=$1 AND NOT EXISTS(SELECT 1 FROM metric_snapshots m WHERE m.submission_id=cs.id AND m.captured_at=$1) ORDER BY cs.id LIMIT 200`, bucket)
	if err != nil {
		return result, err
	}
	type work struct {
		id, platform, campaignID string
		submitted                time.Time
	}
	batch := []work{}
	for rows.Next() {
		var w work
		if err := rows.Scan(&w.id, &w.platform, &w.submitted, &w.campaignID); err != nil {
			rows.Close()
			return result, err
		}
		batch = append(batch, w)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	for _, w := range batch {
		m, err := MockMetrics(w.id, w.platform, w.submitted, bucket)
		if err != nil {
			return result, err
		}
		// Recheck approval under the row lock used by moderation. A concurrent rejection
		// cannot produce a newly collected snapshot after the moderation transaction.
		tx, err := c.pool.Begin(ctx)
		if err != nil {
			return result, err
		}
		var campaignStatus string
		if err = tx.QueryRow(ctx, `SELECT status FROM campaigns WHERE id=$1 FOR UPDATE`, w.campaignID).Scan(&campaignStatus); err != nil {
			_ = tx.Rollback(ctx)
			return result, err
		}
		if campaignStatus != "ACTIVE" && campaignStatus != "PAUSED" && campaignStatus != "COMPLETED" {
			_ = tx.Rollback(ctx)
			continue
		}
		var approved bool
		err = tx.QueryRow(ctx, `SELECT status='APPROVED' FROM clip_submissions WHERE id=$1 FOR UPDATE`, w.id).Scan(&approved)
		if err != nil {
			_ = tx.Rollback(ctx)
			return result, err
		}
		if !approved {
			_ = tx.Rollback(ctx)
			continue
		}
		var previous Metrics
		if err = tx.QueryRow(ctx, `SELECT COALESCE(max(views_count),0),COALESCE(max(likes_count),0),COALESCE(max(comments_count),0),COALESCE(max(shares_count),0),COALESCE(max(saves_count),0) FROM metric_snapshots WHERE submission_id=$1`, w.id).Scan(&previous.Views, &previous.Likes, &previous.Comments, &previous.Shares, &previous.Saves); err != nil {
			_ = tx.Rollback(ctx)
			return result, err
		}
		m.Views = max(m.Views, previous.Views)
		m.Likes = max(m.Likes, previous.Likes)
		m.Comments = max(m.Comments, previous.Comments)
		m.Shares = max(m.Shares, previous.Shares)
		m.Saves = max(m.Saves, previous.Saves)
		tag, err := tx.Exec(ctx, `INSERT INTO metric_snapshots(submission_id,captured_at,views_count,likes_count,comments_count,shares_count,saves_count) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(submission_id,captured_at) DO NOTHING`, w.id, bucket, m.Views, m.Likes, m.Comments, m.Shares, m.Saves)
		if err != nil {
			_ = tx.Rollback(ctx)
			return result, err
		}
		if err = tx.Commit(ctx); err != nil {
			return result, err
		}
		result.Collected += tag.RowsAffected()
	}
	if result.Collected > 0 {
		_ = c.redis.Incr(ctx, "analytics:generation").Err()
	}
	return result, nil
}

func (c *Collector) Run(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		result, err := c.Collect(ctx)
		if err != nil && ctx.Err() == nil {
			logger.Warn("simulated metric collection failed", "error", err)
		} else if result.Collected > 0 {
			logger.Info("simulated metric collection completed", "count", result.Collected)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (c *Collector) String() string { return fmt.Sprint("simulated social metrics collector") }

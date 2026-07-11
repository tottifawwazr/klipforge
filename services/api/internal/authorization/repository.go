package authorization

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errResourceMissing = errors.New("authorization resource missing")

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

type campaignRecord struct{ BrandID, Status string }

func (r *Repository) campaign(ctx context.Context, id string) (campaignRecord, error) {
	var record campaignRecord
	err := r.pool.QueryRow(ctx, `SELECT brand_id,status FROM campaigns WHERE id=$1`, id).Scan(&record.BrandID, &record.Status)
	return record, resourceError("load campaign authorization record", err)
}

type participationRecord struct{ ClipperID, BrandID, CampaignStatus string }

func (r *Repository) participation(ctx context.Context, id string) (participationRecord, error) {
	var record participationRecord
	err := r.pool.QueryRow(ctx, `SELECT cp.clipper_id,c.brand_id,c.status
		FROM campaign_participants cp JOIN campaigns c ON c.id=cp.campaign_id WHERE cp.id=$1`, id).
		Scan(&record.ClipperID, &record.BrandID, &record.CampaignStatus)
	return record, resourceError("load participation authorization record", err)
}

type submissionRecord struct{ ClipperID, BrandID, Status string }

func (r *Repository) submission(ctx context.Context, id string) (submissionRecord, error) {
	var record submissionRecord
	err := r.pool.QueryRow(ctx, `SELECT cp.clipper_id,c.brand_id,cs.status
		FROM clip_submissions cs
		JOIN campaign_participants cp ON cp.id=cs.participant_id
		JOIN campaigns c ON c.id=cs.campaign_id
		WHERE cs.id=$1`, id).Scan(&record.ClipperID, &record.BrandID, &record.Status)
	return record, resourceError("load submission authorization record", err)
}

type payoutRecord struct{ ClipperID, BrandID string }

func (r *Repository) payout(ctx context.Context, id string) (payoutRecord, error) {
	var record payoutRecord
	err := r.pool.QueryRow(ctx, `SELECT cp.clipper_id,c.brand_id
		FROM payouts p
		JOIN campaign_participants cp ON cp.id=p.participant_id
		JOIN campaigns c ON c.id=p.campaign_id
		WHERE p.id=$1`, id).Scan(&record.ClipperID, &record.BrandID)
	return record, resourceError("load payout authorization record", err)
}

func (r *Repository) profileExists(ctx context.Context, userID string) error {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM profiles WHERE user_id=$1)`, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("load profile authorization record: %w", err)
	}
	if !exists {
		return errResourceMissing
	}
	return nil
}

func resourceError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errResourceMissing
	}
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return nil
}

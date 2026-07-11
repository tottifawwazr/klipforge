package campaign

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klipforge/klipforge/services/api/internal/auth"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const campaignColumns = `id,brand_id,name,slug,description,brief,status,budget_amount,remaining_budget_amount,cpm_amount,max_payout_per_clip,currency_code,start_date,end_date,thumbnail_url,created_at,updated_at`

func (r *Repository) Create(ctx context.Context, c Campaign, actor auth.Principal, requestID, ip, userAgent string) (Campaign, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Campaign{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	err = tx.QueryRow(ctx, `INSERT INTO campaigns (brand_id,name,slug,description,brief,status,budget_amount,remaining_budget_amount,cpm_amount,max_payout_per_clip,currency_code,start_date,end_date,thumbnail_url)
		VALUES ($1,$2,$3,$4,$5,'DRAFT',$6,$6,$7,$8,'USD',$9,$10,$11) RETURNING `+campaignColumns,
		c.BrandID, c.Title, c.Slug, c.Description, c.Brief, c.TotalBudget, c.CPMRate, c.MaximumPayoutPerClip, c.StartDate, c.EndDate, c.ThumbnailURL).Scan(scanCampaign(&c)...)
	if err != nil {
		return Campaign{}, mapWriteError(err)
	}
	if err = replacePlatforms(ctx, tx, c.ID, c.Platforms); err != nil {
		return Campaign{}, err
	}
	if err = replaceRequirements(ctx, tx, c.ID, c.Requirements); err != nil {
		return Campaign{}, err
	}
	if err = insertAudit(ctx, tx, actor, "campaign.created", c.ID, requestID, ip, userAgent, map[string]any{"status": "DRAFT"}); err != nil {
		return Campaign{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Campaign{}, err
	}
	return r.Get(ctx, c.ID)
}

func (r *Repository) Get(ctx context.Context, id string) (Campaign, error) {
	return loadCampaign(ctx, r.pool, id)
}

func (r *Repository) List(ctx context.Context, q ListQuery) ([]Campaign, Page, error) {
	args := []any{}
	where := []string{"1=1"}
	add := func(value any) string { args = append(args, value); return fmt.Sprintf("$%d", len(args)) }
	if q.Status != "" {
		where = append(where, "c.status="+add(q.Status))
	}
	if q.BrandID != "" {
		where = append(where, "c.brand_id="+add(q.BrandID))
	}
	if q.Platform != "" {
		where = append(where, "EXISTS (SELECT 1 FROM campaign_platforms cp WHERE cp.campaign_id=c.id AND cp.platform="+add(q.Platform)+")")
	}
	if q.Search != "" {
		p := add("%" + q.Search + "%")
		where = append(where, "(c.name ILIKE "+p+" OR c.description ILIKE "+p+" OR c.slug ILIKE "+p+")")
	}
	if q.StartDate != "" {
		where = append(where, "c.start_date >= "+add(q.StartDate))
	}
	if q.EndDate != "" {
		where = append(where, "c.end_date <= "+add(q.EndDate))
	}
	filter := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM campaigns c WHERE "+filter, args...).Scan(&total); err != nil {
		return nil, Page{}, err
	}
	sort := map[string]string{"created_at": "c.created_at", "start_date": "c.start_date", "end_date": "c.end_date", "title": "c.name"}[q.Sort]
	if sort == "" {
		sort = "c.created_at"
	}
	direction := "DESC"
	if q.Direction == "asc" {
		direction = "ASC"
	}
	args = append(args, q.Limit, (q.Page-1)*q.Limit)
	query := `SELECT ` + campaignColumns + ` FROM campaigns c WHERE ` + filter + ` ORDER BY ` + sort + ` ` + direction + `, c.id ASC LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, Page{}, err
	}
	defer rows.Close()
	items := []Campaign{}
	for rows.Next() {
		var c Campaign
		if err = rows.Scan(scanCampaign(&c)...); err != nil {
			return nil, Page{}, err
		}
		loaded, err := loadCampaignDetails(ctx, r.pool, c)
		if err != nil {
			return nil, Page{}, err
		}
		items = append(items, loaded)
	}
	if err = rows.Err(); err != nil {
		return nil, Page{}, err
	}
	pages := int(math.Ceil(float64(total) / float64(q.Limit)))
	return items, Page{q.Page, q.Limit, total, pages}, nil
}

func (r *Repository) Update(ctx context.Context, c Campaign, platforms *[]string, requirements *[]Requirement, actor auth.Principal, requestID, ip, userAgent string) (Campaign, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Campaign{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `UPDATE campaigns SET name=$2,description=$3,brief=$4,start_date=$5,end_date=$6,thumbnail_url=$7 WHERE id=$1`, c.ID, c.Title, c.Description, c.Brief, c.StartDate, c.EndDate, c.ThumbnailURL)
	if err != nil {
		return Campaign{}, err
	}
	if platforms != nil {
		if err = replacePlatforms(ctx, tx, c.ID, *platforms); err != nil {
			return Campaign{}, err
		}
	}
	if requirements != nil {
		if err = replaceRequirements(ctx, tx, c.ID, *requirements); err != nil {
			return Campaign{}, err
		}
	}
	if err = insertAudit(ctx, tx, actor, "campaign.updated", c.ID, requestID, ip, userAgent, map[string]any{"status": c.Status}); err != nil {
		return Campaign{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Campaign{}, err
	}
	return r.Get(ctx, c.ID)
}

func (r *Repository) Transition(ctx context.Context, id, from, to, action string, actor auth.Principal, requestID, ip, userAgent string) (Campaign, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Campaign{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	command, err := tx.Exec(ctx, `UPDATE campaigns SET status=$3 WHERE id=$1 AND status=$2`, id, from, to)
	if err != nil {
		return Campaign{}, err
	}
	if command.RowsAffected() != 1 {
		return Campaign{}, ErrInvalidTransition
	}
	if err = insertAudit(ctx, tx, actor, action, id, requestID, ip, userAgent, map[string]any{"previous_status": from, "status": to}); err != nil {
		return Campaign{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Campaign{}, err
	}
	return r.Get(ctx, id)
}

type campaignQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadCampaign(ctx context.Context, db campaignQuerier, id string) (Campaign, error) {
	var c Campaign
	err := db.QueryRow(ctx, `SELECT `+campaignColumns+` FROM campaigns WHERE id=$1`, id).Scan(scanCampaign(&c)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, ErrNotFound
	}
	if err != nil {
		return Campaign{}, err
	}
	return loadCampaignDetails(ctx, db, c)
}
func loadCampaignDetails(ctx context.Context, db interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, c Campaign) (Campaign, error) {
	rows, err := db.Query(ctx, `SELECT platform FROM campaign_platforms WHERE campaign_id=$1 ORDER BY platform`, c.ID)
	if err != nil {
		return Campaign{}, err
	}
	for rows.Next() {
		var platform string
		if err = rows.Scan(&platform); err != nil {
			rows.Close()
			return Campaign{}, err
		}
		c.Platforms = append(c.Platforms, platform)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return Campaign{}, err
	}
	rows.Close()
	rows, err = db.Query(ctx, `SELECT requirement_type,description,position FROM campaign_requirements WHERE campaign_id=$1 ORDER BY position`, c.ID)
	if err != nil {
		return Campaign{}, err
	}
	for rows.Next() {
		var requirement Requirement
		if err = rows.Scan(&requirement.Type, &requirement.Description, &requirement.Position); err != nil {
			rows.Close()
			return Campaign{}, err
		}
		c.Requirements = append(c.Requirements, requirement)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return Campaign{}, err
	}
	rows.Close()
	return c, nil
}
func scanCampaign(c *Campaign) []any {
	return []any{&c.ID, &c.BrandID, &c.Title, &c.Slug, &c.Description, &c.Brief, &c.Status, &c.TotalBudget, &c.RemainingBudget, &c.CPMRate, &c.MaximumPayoutPerClip, &c.CurrencyCode, &c.StartDate, &c.EndDate, &c.ThumbnailURL, &c.CreatedAt, &c.UpdatedAt}
}

func replacePlatforms(ctx context.Context, tx pgx.Tx, campaignID string, platforms []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM campaign_platforms WHERE campaign_id=$1`, campaignID); err != nil {
		return err
	}
	for _, platform := range platforms {
		if _, err := tx.Exec(ctx, `INSERT INTO campaign_platforms(campaign_id,platform) VALUES($1,$2)`, campaignID, platform); err != nil {
			return err
		}
	}
	return nil
}
func replaceRequirements(ctx context.Context, tx pgx.Tx, campaignID string, requirements []Requirement) error {
	if _, err := tx.Exec(ctx, `DELETE FROM campaign_requirements WHERE campaign_id=$1`, campaignID); err != nil {
		return err
	}
	for index, requirement := range requirements {
		if _, err := tx.Exec(ctx, `INSERT INTO campaign_requirements(campaign_id,requirement_type,description,position) VALUES($1,$2,$3,$4)`, campaignID, requirement.Type, requirement.Description, index+1); err != nil {
			return err
		}
	}
	return nil
}
func insertAudit(ctx context.Context, tx pgx.Tx, actor auth.Principal, action, campaignID, requestID, ip, userAgent string, metadata map[string]any) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_logs(actor_user_id,actor_email,actor_role,action,entity_type,entity_id,request_id,metadata,ip_address,user_agent) VALUES($1,$2,$3,$4,'campaign',$5,NULLIF($6,''),$7,NULLIF($8,'')::inet,NULLIF($9,''))`, actor.UserID, actor.Email, actor.Role, action, campaignID, requestID, payload, ip, userAgent)
	return err
}
func mapWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrSlugConflict
	}
	return err
}

package participation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klipforge/klipforge/services/api/internal/auth"
	"math"
	"strings"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool} }
func (r *Repository) ListParticipations(ctx context.Context, clipperID, campaignID, brandID string, q ListQuery) ([]Participation, Page, error) {
	args := []any{}
	where := []string{"1=1"}
	add := func(v any) string { args = append(args, v); return fmt.Sprintf("$%d", len(args)) }
	if clipperID != "" {
		where = append(where, "cp.clipper_id="+add(clipperID))
	}
	if campaignID != "" {
		where = append(where, "cp.campaign_id="+add(campaignID))
	}
	if brandID != "" {
		where = append(where, "c.brand_id="+add(brandID))
	}
	if q.Status != "" {
		where = append(where, "cp.status="+add(q.Status))
	}
	filter := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM campaign_participants cp JOIN campaigns c ON c.id=cp.campaign_id WHERE `+filter, args...).Scan(&total); err != nil {
		return nil, Page{}, err
	}
	sort := map[string]string{"created_at": "cp.created_at", "joined_at": "cp.joined_at", "status": "cp.status"}[q.Sort]
	if sort == "" {
		sort = "cp.created_at"
	}
	direction := "DESC"
	if q.Direction == "asc" {
		direction = "ASC"
	}
	args = append(args, q.Limit, (q.Page-1)*q.Limit)
	rows, err := r.pool.Query(ctx, `SELECT cp.id,cp.campaign_id,cp.clipper_id,cp.status,cp.joined_at,cp.created_at FROM campaign_participants cp JOIN campaigns c ON c.id=cp.campaign_id WHERE `+filter+` ORDER BY `+sort+` `+direction+`,cp.id ASC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, Page{}, err
	}
	defer rows.Close()
	out := []Participation{}
	for rows.Next() {
		var p Participation
		if err = rows.Scan(&p.ID, &p.CampaignID, &p.ClipperID, &p.Status, &p.JoinedAt, &p.CreatedAt); err != nil {
			return nil, Page{}, err
		}
		out = append(out, p)
	}
	pages := int(math.Ceil(float64(total) / float64(q.Limit)))
	return out, Page{q.Page, q.Limit, total, pages}, rows.Err()
}
func (r *Repository) Join(ctx context.Context, campaignID string, actor auth.Principal, requestID, ip, agent string) (Participation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Participation{}, err
	}
	defer tx.Rollback(ctx)
	var p Participation
	err = tx.QueryRow(ctx, `INSERT INTO campaign_participants(campaign_id,clipper_id,status,joined_at) VALUES($1,$2,'ACCEPTED',NOW()) RETURNING id,campaign_id,clipper_id,status,joined_at,created_at`, campaignID, actor.UserID).Scan(&p.ID, &p.CampaignID, &p.ClipperID, &p.Status, &p.JoinedAt, &p.CreatedAt)
	if err != nil {
		return Participation{}, mapWrite(err, ErrAlreadyJoined)
	}
	if err = audit(ctx, tx, actor, "campaign.joined", "campaign_participation", p.ID, requestID, ip, agent, map[string]any{"campaign_id": campaignID}); err != nil {
		return Participation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Participation{}, err
	}
	return p, nil
}
func (r *Repository) Participation(ctx context.Context, campaignID, clipperID string) (Participation, error) {
	var p Participation
	err := r.pool.QueryRow(ctx, `SELECT id,campaign_id,clipper_id,status,joined_at,created_at FROM campaign_participants WHERE campaign_id=$1 AND clipper_id=$2`, campaignID, clipperID).Scan(&p.ID, &p.CampaignID, &p.ClipperID, &p.Status, &p.JoinedAt, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Participation{}, ErrParticipationNotFound
	}
	return p, err
}

func (r *Repository) ParticipationByID(ctx context.Context, id string) (Participation, error) {
	var p Participation
	err := r.pool.QueryRow(ctx, `SELECT id,campaign_id,clipper_id,status,joined_at,created_at FROM campaign_participants WHERE id=$1`, id).Scan(&p.ID, &p.CampaignID, &p.ClipperID, &p.Status, &p.JoinedAt, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Participation{}, ErrParticipationNotFound
	}
	return p, err
}
func (r *Repository) CreateSubmission(ctx context.Context, s Submission, actor auth.Principal, requestID, ip, agent string) (Submission, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Submission{}, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `INSERT INTO clip_submissions(campaign_id,participant_id,content_url,platform,caption,status) VALUES($1,$2,$3,$4,$5,'PENDING') RETURNING id,campaign_id,participant_id,content_url,platform,caption,status,submitted_at,created_at,updated_at,reviewed_at,CASE WHEN status='REJECTED' THEN review_note END`, s.CampaignID, s.ParticipantID, s.ContentURL, s.Platform, s.Caption).Scan(&s.ID, &s.CampaignID, &s.ParticipantID, &s.ContentURL, &s.Platform, &s.Caption, &s.Status, &s.SubmittedAt, &s.CreatedAt, &s.UpdatedAt, &s.ReviewedAt, &s.RejectionReason)
	if err != nil {
		return Submission{}, mapWrite(err, ErrDuplicateURL)
	}
	if err = audit(ctx, tx, actor, "submission.created", "clip_submission", s.ID, requestID, ip, agent, map[string]any{"campaign_id": s.CampaignID, "platform": s.Platform}); err != nil {
		return Submission{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Submission{}, err
	}
	return s, nil
}
func (r *Repository) Submission(ctx context.Context, id string) (Submission, string, string, error) {
	var s Submission
	var clipper, brand string
	err := r.pool.QueryRow(ctx, `SELECT cs.id,cs.campaign_id,cs.participant_id,cs.content_url,cs.platform,cs.caption,cs.status,cs.submitted_at,cs.created_at,cs.updated_at,cs.reviewed_at,CASE WHEN cs.status='REJECTED' THEN cs.review_note END,cp.clipper_id,c.brand_id FROM clip_submissions cs JOIN campaign_participants cp ON cp.id=cs.participant_id JOIN campaigns c ON c.id=cs.campaign_id WHERE cs.id=$1`, id).Scan(&s.ID, &s.CampaignID, &s.ParticipantID, &s.ContentURL, &s.Platform, &s.Caption, &s.Status, &s.SubmittedAt, &s.CreatedAt, &s.UpdatedAt, &s.ReviewedAt, &s.RejectionReason, &clipper, &brand)
	if errors.Is(err, pgx.ErrNoRows) {
		return Submission{}, "", "", ErrSubmissionNotFound
	}
	return s, clipper, brand, err
}
func (r *Repository) UpdateSubmission(ctx context.Context, s Submission, actor auth.Principal, requestID, ip, agent string) (Submission, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Submission{}, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `UPDATE clip_submissions SET content_url=$2,platform=$3,caption=$4 WHERE id=$1 AND status='PENDING' RETURNING id,campaign_id,participant_id,content_url,platform,caption,status,submitted_at,created_at,updated_at,reviewed_at,CASE WHEN status='REJECTED' THEN review_note END`, s.ID, s.ContentURL, s.Platform, s.Caption).Scan(&s.ID, &s.CampaignID, &s.ParticipantID, &s.ContentURL, &s.Platform, &s.Caption, &s.Status, &s.SubmittedAt, &s.CreatedAt, &s.UpdatedAt, &s.ReviewedAt, &s.RejectionReason)
	if errors.Is(err, pgx.ErrNoRows) {
		return Submission{}, ErrNotEditable
	}
	if err != nil {
		return Submission{}, mapWrite(err, ErrDuplicateURL)
	}
	if err = audit(ctx, tx, actor, "submission.updated", "clip_submission", s.ID, requestID, ip, agent, map[string]any{"platform": s.Platform}); err != nil {
		return Submission{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Submission{}, err
	}
	return s, nil
}
func (r *Repository) ListSubmissions(ctx context.Context, clipperID, brandID, campaignID string, q ListQuery) ([]Submission, Page, error) {
	args := []any{}
	where := []string{"1=1"}
	add := func(v any) string { args = append(args, v); return fmt.Sprintf("$%d", len(args)) }
	if clipperID != "" {
		where = append(where, "cp.clipper_id="+add(clipperID))
	}
	if brandID != "" {
		where = append(where, "c.brand_id="+add(brandID))
	}
	if campaignID != "" {
		where = append(where, "cs.campaign_id="+add(campaignID))
	}
	if q.Platform != "" {
		where = append(where, "cs.platform="+add(q.Platform))
	}
	if q.Status != "" {
		where = append(where, "cs.status="+add(q.Status))
	}
	filter := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM clip_submissions cs JOIN campaign_participants cp ON cp.id=cs.participant_id JOIN campaigns c ON c.id=cs.campaign_id WHERE `+filter, args...).Scan(&total); err != nil {
		return nil, Page{}, err
	}
	sort := map[string]string{"submitted_at": "cs.submitted_at", "created_at": "cs.created_at", "updated_at": "cs.updated_at", "status": "cs.status"}[q.Sort]
	if sort == "" {
		sort = "cs.submitted_at"
	}
	dir := "DESC"
	if q.Direction == "asc" {
		dir = "ASC"
	}
	args = append(args, q.Limit, (q.Page-1)*q.Limit)
	rows, err := r.pool.Query(ctx, `SELECT cs.id,cs.campaign_id,cs.participant_id,cs.content_url,cs.platform,cs.caption,cs.status,cs.submitted_at,cs.created_at,cs.updated_at,cs.reviewed_at,CASE WHEN cs.status='REJECTED' THEN cs.review_note END FROM clip_submissions cs JOIN campaign_participants cp ON cp.id=cs.participant_id JOIN campaigns c ON c.id=cs.campaign_id WHERE `+filter+` ORDER BY `+sort+` `+dir+`,cs.id ASC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, Page{}, err
	}
	defer rows.Close()
	out := []Submission{}
	for rows.Next() {
		var s Submission
		if err = rows.Scan(&s.ID, &s.CampaignID, &s.ParticipantID, &s.ContentURL, &s.Platform, &s.Caption, &s.Status, &s.SubmittedAt, &s.CreatedAt, &s.UpdatedAt, &s.ReviewedAt, &s.RejectionReason); err != nil {
			return nil, Page{}, err
		}
		out = append(out, s)
	}
	if err = rows.Err(); err != nil {
		return nil, Page{}, err
	}
	pages := int(math.Ceil(float64(total) / float64(q.Limit)))
	return out, Page{q.Page, q.Limit, total, pages}, nil
}
func audit(ctx context.Context, tx pgx.Tx, a auth.Principal, action, typ, id, requestID, ip, agent string, meta map[string]any) error {
	payload, _ := json.Marshal(meta)
	_, err := tx.Exec(ctx, `INSERT INTO audit_logs(actor_user_id,actor_email,actor_role,action,entity_type,entity_id,request_id,metadata,ip_address,user_agent) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,''),$8,NULLIF($9,'')::inet,NULLIF($10,''))`, a.UserID, a.Email, a.Role, action, typ, id, requestID, payload, ip, agent)
	return err
}
func mapWrite(err error, dup *Error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return dup
	}
	return err
}

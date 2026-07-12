package moderation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/klipforge/klipforge/services/api/internal/participation"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const submissionColumns = `cs.id,cs.campaign_id,c.name,cs.participant_id,cp.clipper_id,p.display_name,cs.platform,cs.content_url,cs.caption,cs.status,cs.review_note,cs.reviewed_by_user_id,cs.submitted_at,cs.reviewed_at,cs.created_at,cs.updated_at,c.brand_id`

func (r *Repository) CampaignOwner(ctx context.Context, campaignID string) (string, error) {
	var owner string
	err := r.pool.QueryRow(ctx, `SELECT brand_id FROM campaigns WHERE id=$1`, campaignID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrCampaignNotFound
	}
	return owner, err
}

func (r *Repository) List(ctx context.Context, query QueueQuery) ([]Submission, Page, error) {
	args := []any{}
	where := []string{"1=1"}
	add := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if query.CampaignID != "" {
		where = append(where, "cs.campaign_id="+add(query.CampaignID))
	}
	if query.BrandID != "" {
		where = append(where, "c.brand_id="+add(query.BrandID))
	}
	if query.Status != "" {
		where = append(where, "cs.status="+add(query.Status))
	} else if query.UnresolvedOnly {
		where = append(where, "cs.status IN ('PENDING','FLAGGED')")
	}
	if query.Platform != "" {
		where = append(where, "cs.platform="+add(query.Platform))
	}
	if query.Search != "" {
		pattern := add("%" + query.Search + "%")
		where = append(where, "(p.display_name ILIKE "+pattern+" OR cs.content_url ILIKE "+pattern+" OR cs.caption ILIKE "+pattern+")")
	}
	if query.SubmittedFrom != "" {
		where = append(where, "cs.submitted_at >= "+add(query.SubmittedFrom)+"::date")
	}
	if query.SubmittedTo != "" {
		where = append(where, "cs.submitted_at < "+add(query.SubmittedTo)+"::date + INTERVAL '1 day'")
	}
	if query.ReviewedFrom != "" {
		where = append(where, "cs.reviewed_at >= "+add(query.ReviewedFrom)+"::date")
	}
	if query.ReviewedTo != "" {
		where = append(where, "cs.reviewed_at < "+add(query.ReviewedTo)+"::date + INTERVAL '1 day'")
	}
	filter := strings.Join(where, " AND ")
	joins := ` FROM clip_submissions cs JOIN campaign_participants cp ON cp.id=cs.participant_id JOIN campaigns c ON c.id=cs.campaign_id JOIN profiles p ON p.user_id=cp.clipper_id`
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*)`+joins+` WHERE `+filter, args...).Scan(&total); err != nil {
		return nil, Page{}, err
	}
	sort := map[string]string{
		"submitted_at": "cs.submitted_at",
		"reviewed_at":  "cs.reviewed_at",
		"created_at":   "cs.created_at",
		"status":       "cs.status",
		"platform":     "cs.platform",
	}[query.Sort]
	direction := "ASC"
	if query.Direction == "desc" {
		direction = "DESC"
	}
	args = append(args, query.Limit, (query.Page-1)*query.Limit)
	rows, err := r.pool.Query(ctx, `SELECT `+submissionColumns+joins+` WHERE `+filter+` ORDER BY `+sort+` `+direction+` NULLS LAST,cs.id ASC LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, Page{}, err
	}
	defer rows.Close()
	items := []Submission{}
	for rows.Next() {
		var submission Submission
		if err = rows.Scan(scanSubmission(&submission)...); err != nil {
			return nil, Page{}, err
		}
		items = append(items, submission)
	}
	if err = rows.Err(); err != nil {
		return nil, Page{}, err
	}
	totalPages := int(math.Ceil(float64(total) / float64(query.Limit)))
	return items, Page{Page: query.Page, Limit: query.Limit, TotalItems: total, TotalPages: totalPages}, nil
}

func (r *Repository) Get(ctx context.Context, submissionID string) (Submission, error) {
	submission, err := loadSubmission(ctx, r.pool, submissionID, false)
	if err != nil {
		return Submission{}, err
	}
	submission.Requirements, err = loadRequirements(ctx, r.pool, submission.CampaignID)
	return submission, err
}

func (r *Repository) Moderate(ctx context.Context, actor auth.Principal, campaignID, submissionID, target string, input ActionInput) (Submission, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Submission{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	submission, err := loadSubmission(ctx, tx, submissionID, true)
	if err != nil {
		return Submission{}, err
	}
	if campaignID != "" && submission.CampaignID != campaignID || actor.Role == auth.RoleBrand && submission.BrandID != actor.UserID {
		return Submission{}, ErrSubmissionNotFound
	}
	if submission.Status == StatusApproved || submission.Status == StatusRejected {
		return Submission{}, ErrAlreadyReviewed
	}
	if !validTransition(submission.Status, target) {
		return Submission{}, ErrInvalidTransition
	}
	canonicalURL, validationErr := participation.CanonicalContentURL(submission.ContentURL, submission.Platform)
	if validationErr != nil || canonicalURL != submission.ContentURL || submission.ParticipantID == "" {
		return Submission{}, ErrSubmissionNotReady
	}

	previousStatus := submission.Status
	var reason *string
	if input.Reason != "" {
		reason = &input.Reason
	}
	command, err := tx.Exec(ctx, `UPDATE clip_submissions SET status=$2,reviewed_by_user_id=$3,reviewed_at=NOW(),review_note=$4 WHERE id=$1 AND status=$5`, submissionID, target, actor.UserID, reason, previousStatus)
	if err != nil {
		return Submission{}, err
	}
	if command.RowsAffected() != 1 {
		return Submission{}, ErrModerationConflict
	}

	action := moderationAction(previousStatus, target)
	metadata := map[string]any{"campaign_id": submission.CampaignID, "previous_status": previousStatus, "status": target}
	if reason != nil {
		metadata["reason"] = *reason
	}
	if err = insertAudit(ctx, tx, actor, action, submissionID, input, metadata); err != nil {
		return Submission{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Submission{}, err
	}
	return r.Get(ctx, submissionID)
}

type submissionQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadSubmission(ctx context.Context, db submissionQuerier, submissionID string, lock bool) (Submission, error) {
	query := `SELECT ` + submissionColumns + ` FROM clip_submissions cs JOIN campaign_participants cp ON cp.id=cs.participant_id JOIN campaigns c ON c.id=cs.campaign_id JOIN profiles p ON p.user_id=cp.clipper_id WHERE cs.id=$1`
	if lock {
		query += ` FOR UPDATE OF cs`
	}
	var submission Submission
	err := db.QueryRow(ctx, query, submissionID).Scan(scanSubmission(&submission)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return Submission{}, ErrSubmissionNotFound
	}
	return submission, err
}

type requirementQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadRequirements(ctx context.Context, db requirementQuerier, campaignID string) ([]Requirement, error) {
	rows, err := db.Query(ctx, `SELECT requirement_type,description,position FROM campaign_requirements WHERE campaign_id=$1 ORDER BY position`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	requirements := []Requirement{}
	for rows.Next() {
		var requirement Requirement
		if err = rows.Scan(&requirement.Type, &requirement.Description, &requirement.Position); err != nil {
			return nil, err
		}
		requirements = append(requirements, requirement)
	}
	return requirements, rows.Err()
}

func scanSubmission(submission *Submission) []any {
	return []any{
		&submission.ID, &submission.CampaignID, &submission.CampaignTitle, &submission.ParticipantID,
		&submission.ClipperID, &submission.ClipperDisplayName, &submission.Platform, &submission.ContentURL,
		&submission.Caption, &submission.Status, &submission.Reason, &submission.ReviewedByUserID,
		&submission.SubmittedAt, &submission.ReviewedAt, &submission.CreatedAt, &submission.UpdatedAt, &submission.BrandID,
	}
}

func moderationAction(previousStatus, target string) string {
	if previousStatus == StatusFlagged {
		return "submission.flag_resolved"
	}
	switch target {
	case StatusApproved:
		return "submission.approved"
	case StatusRejected:
		return "submission.rejected"
	default:
		return "submission.flagged"
	}
}

func insertAudit(ctx context.Context, tx pgx.Tx, actor auth.Principal, action, submissionID string, input ActionInput, metadata map[string]any) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO audit_logs(actor_user_id,actor_email,actor_role,action,entity_type,entity_id,request_id,metadata,ip_address,user_agent) VALUES($1,$2,$3,$4,'clip_submission',$5,NULLIF($6,''),$7,NULLIF($8,'')::inet,NULLIF($9,''))`, actor.UserID, actor.Email, actor.Role, action, submissionID, input.RequestID, payload, input.IPAddress, input.UserAgent)
	return err
}

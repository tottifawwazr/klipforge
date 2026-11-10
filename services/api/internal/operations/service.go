// Package operations implements audited payout records and user notifications.
// Processing records an externally completed payment; it never transfers money.
package operations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/klipforge/klipforge/services/api/internal/auth"
	"regexp"
	"strings"
	"time"
)

type Error struct {
	Code, Message string
	Status        int
}

func (e *Error) Error() string { return e.Code }

var (
	ErrForbidden = &Error{"FORBIDDEN", "You do not have access to this operation.", 403}
	ErrNotFound  = &Error{"RESOURCE_NOT_FOUND", "The resource was not found.", 404}
	ErrInvalid   = &Error{"VALIDATION_ERROR", "The request is invalid.", 400}
	ErrKey       = &Error{"IDEMPOTENCY_CONFLICT", "This idempotency key was already used for a different request.", 409}
	ErrState     = &Error{"PAYOUT_STATE_CONFLICT", "This payout cannot make that transition.", 409}
	ErrCampaign  = &Error{"CAMPAIGN_NOT_COMPLETED", "Complete the campaign before calculating final payouts.", 409}
	ErrMetrics   = &Error{"PAYOUT_NOT_READY", "An approved submission with collected metrics and a positive payout is required.", 409}
	ErrBudget    = &Error{"INSUFFICIENT_BUDGET", "The campaign has insufficient unreserved budget.", 409}
	ErrQuote     = &Error{"PAYOUT_QUOTE_CHANGED", "The payout amount changed. Preview it again before confirming.", 409}
	ErrDuplicate = &Error{"PAYOUT_ALREADY_EXISTS", "A payout record already exists for this submission or payment reference.", 409}
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool} }
func active(a auth.Principal) bool {
	return a.UserID != "" && a.AccountActive && a.SessionActive && (a.Role == auth.RoleAdmin || a.Role == auth.RoleBrand || a.Role == auth.RoleClipper)
}
func validID(id string) bool {
	b := strings.ReplaceAll(id, "-", "")
	_, err := hex.DecodeString(b)
	return len(id) == 36 && len(b) == 32 && err == nil && id[8] == '-' && id[13] == '-' && id[18] == '-' && id[23] == '-'
}

var keyPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{16,100}$`)
var referencePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:/-]{2,127}$`)

type Payout struct {
	ID            string     `json:"id"`
	CampaignID    string     `json:"campaign_id"`
	CampaignTitle string     `json:"campaign_title"`
	SubmissionID  string     `json:"submission_id"`
	ParticipantID string     `json:"participant_id"`
	DisplayName   string     `json:"display_name"`
	Status        string     `json:"status"`
	Amount        string     `json:"amount"`
	Currency      string     `json:"currency_code"`
	BasisViews    int64      `json:"basis_views"`
	BasisAt       *time.Time `json:"basis_snapshot_at"`
	Reference     *string    `json:"processor_reference"`
	CreatedAt     time.Time  `json:"created_at"`
}

const payoutSelect = `SELECT p.id,p.campaign_id,c.name,p.submission_id,p.participant_id,pr.display_name,p.status,p.amount::text,p.currency_code,p.basis_views,p.basis_snapshot_at,p.processor_reference,p.created_at FROM payouts p JOIN campaigns c ON c.id=p.campaign_id JOIN campaign_participants cp ON cp.id=p.participant_id JOIN profiles pr ON pr.user_id=cp.clipper_id `

func scan(row pgx.Row) (Payout, error) {
	var p Payout
	err := row.Scan(&p.ID, &p.CampaignID, &p.CampaignTitle, &p.SubmissionID, &p.ParticipantID, &p.DisplayName, &p.Status, &p.Amount, &p.Currency, &p.BasisViews, &p.BasisAt, &p.Reference, &p.CreatedAt)
	return p, err
}
func (s *Service) List(ctx context.Context, a auth.Principal, page int) ([]Payout, error) {
	if !active(a) {
		return nil, ErrForbidden
	}
	if page < 1 || page > 100000 {
		return nil, ErrInvalid
	}
	rows, err := s.pool.Query(ctx, payoutSelect+`WHERE ($1='ADMIN' OR ($1='BRAND' AND c.brand_id::text=$2) OR ($1='CLIPPER' AND cp.clipper_id::text=$2)) ORDER BY p.created_at DESC,p.id LIMIT 50 OFFSET $3`, a.Role, a.UserID, (page-1)*50)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Payout{}
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
func (s *Service) Create(ctx context.Context, a auth.Principal, submissionID, key string, expected ...string) (Payout, bool, error) {
	if !active(a) || a.Role != auth.RoleAdmin {
		return Payout{}, false, ErrForbidden
	}
	if !validID(submissionID) || !keyPattern.MatchString(key) {
		return Payout{}, false, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Payout{}, false, err
	}
	defer tx.Rollback(ctx)
	expectedAmount := ""
	if len(expected) > 0 {
		expectedAmount = expected[0]
	}
	sum := sha256.Sum256([]byte("payout.create:" + submissionID + ":" + expectedAmount))
	hash := hex.EncodeToString(sum[:])
	_, err = tx.Exec(ctx, `INSERT INTO idempotency_keys(actor_user_id,scope,idempotency_key,request_hash,expires_at)VALUES($1,'payout.create',$2,$3,NOW()+INTERVAL '7 days') ON CONFLICT(actor_user_id,scope,idempotency_key)DO NOTHING`, a.UserID, key, hash)
	if err != nil {
		return Payout{}, false, err
	}
	var keyID, oldHash, status string
	var saved []byte
	if err = tx.QueryRow(ctx, `SELECT id,request_hash,status,response_body FROM idempotency_keys WHERE actor_user_id=$1 AND scope='payout.create' AND idempotency_key=$2 FOR UPDATE`, a.UserID, key).Scan(&keyID, &oldHash, &status, &saved); err != nil {
		return Payout{}, false, err
	}
	if oldHash != hash {
		return Payout{}, false, ErrKey
	}
	if status == "COMPLETED" {
		var p Payout
		if err = json.Unmarshal(saved, &p); err != nil {
			return p, false, err
		}
		return p, true, tx.Commit(ctx)
	}
	var campaignID string
	if err = tx.QueryRow(ctx, `SELECT campaign_id FROM clip_submissions WHERE id=$1`, submissionID).Scan(&campaignID); errors.Is(err, pgx.ErrNoRows) {
		return Payout{}, false, ErrNotFound
	} else if err != nil {
		return Payout{}, false, err
	}
	var campaignStatus string
	if err = tx.QueryRow(ctx, `SELECT status FROM campaigns WHERE id=$1 FOR UPDATE`, campaignID).Scan(&campaignStatus); err != nil {
		return Payout{}, false, err
	}
	if campaignStatus != "COMPLETED" {
		return Payout{}, false, ErrCampaign
	}
	var submissionStatus, participantID string
	if err = tx.QueryRow(ctx, `SELECT status,participant_id FROM clip_submissions WHERE id=$1 FOR UPDATE`, submissionID).Scan(&submissionStatus, &participantID); err != nil {
		return Payout{}, false, err
	}
	if submissionStatus != "APPROVED" {
		return Payout{}, false, ErrMetrics
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM payouts WHERE submission_id=$1)`, submissionID).Scan(&exists); err != nil {
		return Payout{}, false, err
	}
	if exists {
		return Payout{}, false, ErrDuplicate
	}
	var amount, currency string
	var views int64
	var captured time.Time
	err = tx.QueryRow(ctx, `SELECT (floor(least(m.views_count::numeric*c.cpm_amount/1000,c.max_payout_per_clip)*100)/100)::numeric(14,2)::text,c.currency_code,m.views_count,m.captured_at FROM campaigns c JOIN LATERAL(SELECT * FROM metric_snapshots WHERE submission_id=$2 ORDER BY captured_at DESC LIMIT 1)m ON true WHERE c.id=$1 AND c.cpm_amount>0`, campaignID, submissionID).Scan(&amount, &currency, &views, &captured)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && amount == "0.00" {
		return Payout{}, false, ErrMetrics
	}
	if err != nil {
		return Payout{}, false, err
	}
	if len(expected) > 0 && expected[0] != amount {
		return Payout{}, false, ErrQuote
	}
	tag, err := tx.Exec(ctx, `UPDATE campaigns SET remaining_budget_amount=remaining_budget_amount-$2::numeric WHERE id=$1 AND remaining_budget_amount>=$2::numeric`, campaignID, amount)
	if err != nil {
		return Payout{}, false, err
	}
	if tag.RowsAffected() != 1 {
		return Payout{}, false, ErrBudget
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO payouts(campaign_id,submission_id,participant_id,idempotency_key_id,amount,currency_code,basis_views,basis_snapshot_at)VALUES($1,$2,$3,$4,$5::numeric,$6,$7,$8)RETURNING id`, campaignID, submissionID, participantID, keyID, amount, currency, views, captured).Scan(&id)
	if err != nil {
		return Payout{}, false, databaseError(err)
	}
	p, err := scan(tx.QueryRow(ctx, payoutSelect+`WHERE p.id=$1`, id))
	if err != nil {
		return p, false, err
	}
	if err = record(ctx, tx, a, "payout.created", p, "Payout awaiting approval"); err != nil {
		return p, false, err
	}
	payload, err := json.Marshal(p)
	if err != nil {
		return p, false, err
	}
	_, err = tx.Exec(ctx, `UPDATE idempotency_keys SET status='COMPLETED',response_code=201,response_body=$2::jsonb WHERE id=$1`, keyID, payload)
	if err != nil {
		return p, false, err
	}
	return p, false, tx.Commit(ctx)
}
func (s *Service) Transition(ctx context.Context, a auth.Principal, id, action, reference, reason string) (Payout, error) {
	if !active(a) || a.Role != auth.RoleAdmin {
		return Payout{}, ErrForbidden
	}
	if !validID(id) {
		return Payout{}, ErrNotFound
	}
	if action != "approve" && action != "process" && action != "fail" && action != "retry" {
		return Payout{}, ErrInvalid
	}
	reference = strings.TrimSpace(reference)
	reason = strings.TrimSpace(reason)
	if action == "process" && !referencePattern.MatchString(reference) {
		return Payout{}, ErrInvalid
	}
	if action == "fail" && (reason == "" || len(reason) > 1000) {
		return Payout{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Payout{}, err
	}
	defer tx.Rollback(ctx)
	var campaignID string
	if err = tx.QueryRow(ctx, `SELECT campaign_id FROM payouts WHERE id=$1`, id).Scan(&campaignID); errors.Is(err, pgx.ErrNoRows) {
		return Payout{}, ErrNotFound
	} else if err != nil {
		return Payout{}, err
	}
	if _, err = tx.Exec(ctx, `SELECT id FROM campaigns WHERE id=$1 FOR UPDATE`, campaignID); err != nil {
		return Payout{}, err
	}
	p, err := scan(tx.QueryRow(ctx, payoutSelect+`WHERE p.id=$1 FOR UPDATE OF p`, id))
	if err != nil {
		return p, err
	}
	switch action {
	case "retry":
		if p.Status != "FAILED" {
			return p, ErrState
		}
		tag, reserveErr := tx.Exec(ctx, `UPDATE campaigns SET remaining_budget_amount=remaining_budget_amount-$2::numeric WHERE id=$1 AND remaining_budget_amount>=$2::numeric`, campaignID, p.Amount)
		if reserveErr != nil {
			return p, reserveErr
		}
		if tag.RowsAffected() != 1 {
			return p, ErrBudget
		}
		_, err = tx.Exec(ctx, `UPDATE payouts SET status='PENDING',failure_reason=NULL,approved_at=NULL,approved_by_user_id=NULL WHERE id=$1`, id)

	case "approve":
		if p.Status != "PENDING" {
			return p, ErrState
		}
		_, err = tx.Exec(ctx, `UPDATE payouts SET status='APPROVED',approved_by_user_id=$2,approved_at=NOW() WHERE id=$1`, id, a.UserID)
	case "process":
		if p.Status != "APPROVED" {
			return p, ErrState
		}
		_, err = tx.Exec(ctx, `UPDATE payouts SET status='PROCESSED',processor_reference=$2,processed_at=NOW() WHERE id=$1`, id, "manual:"+reference)
	case "fail":
		if p.Status != "PENDING" && p.Status != "APPROVED" {
			return p, ErrState
		}
		_, err = tx.Exec(ctx, `UPDATE payouts SET status='FAILED',failure_reason=$2 WHERE id=$1`, id, reason)
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE campaigns SET remaining_budget_amount=remaining_budget_amount+$2::numeric WHERE id=$1`, campaignID, p.Amount)
		}
	}
	if err != nil {
		return p, databaseError(err)
	}
	p, err = scan(tx.QueryRow(ctx, payoutSelect+`WHERE p.id=$1`, id))
	if err != nil {
		return p, err
	}
	if err = record(ctx, tx, a, "payout."+action, p, "Payout status: "+p.Status); err != nil {
		return p, err
	}
	return p, tx.Commit(ctx)
}
func databaseError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return ErrDuplicate
	}
	return err
}
func record(ctx context.Context, tx pgx.Tx, a auth.Principal, action string, p Payout, title string) error {
	meta, err := json.Marshal(map[string]any{"campaign_id": p.CampaignID, "amount": p.Amount, "currency": p.Currency, "status": p.Status, "basis_views": p.BasisViews, "payment_mode": "MANUAL_RECORD"})
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_logs(actor_user_id,actor_email,actor_role,action,entity_type,entity_id,metadata)VALUES($1,$2,$3,$4,'payout',$5,$6::jsonb)`, a.UserID, a.Email, a.Role, action, p.ID, meta); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO notifications(user_id,notification_type,title,body,data)SELECT clipper_id,'PAYOUT',$2,$3,$4::jsonb FROM campaign_participants WHERE id=$1`, p.ParticipantID, title, fmt.Sprintf("%s %s / %s. KlipForge records payment status; it does not transfer funds.", p.Currency, p.Amount, p.CampaignTitle), meta)
	return err
}

type Notification struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func (s *Service) Notifications(ctx context.Context, a auth.Principal) ([]Notification, error) {
	if !active(a) {
		return nil, ErrForbidden
	}
	rows, err := s.pool.Query(ctx, `SELECT id,title,body,read_at,created_at FROM notifications WHERE user_id=$1 ORDER BY created_at DESC,id LIMIT 100`, a.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Notification{}
	for rows.Next() {
		var n Notification
		if err = rows.Scan(&n.ID, &n.Title, &n.Body, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, n)
	}
	return items, rows.Err()
}
func (s *Service) ReadNotification(ctx context.Context, a auth.Principal, id string) error {
	if !active(a) {
		return ErrForbidden
	}
	if !validID(id) {
		return ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `UPDATE notifications SET read_at=COALESCE(read_at,NOW())WHERE id=$1 AND user_id=$2`, id, a.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

type Audit struct {
	ID         string    `json:"id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   *string   `json:"entity_id"`
	ActorRole  string    `json:"actor_role"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Service) Audit(ctx context.Context, a auth.Principal) ([]Audit, error) {
	if !active(a) || a.Role != auth.RoleAdmin {
		return nil, ErrForbidden
	}
	rows, err := s.pool.Query(ctx, `SELECT id,action,entity_type,entity_id,actor_role,created_at FROM audit_logs ORDER BY created_at DESC,id LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Audit{}
	for rows.Next() {
		var e Audit
		if err = rows.Scan(&e.ID, &e.Action, &e.EntityType, &e.EntityID, &e.ActorRole, &e.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

func (s *Service) Quote(ctx context.Context, a auth.Principal, submissionID string) (Payout, error) {
	if !active(a) || a.Role != auth.RoleAdmin {
		return Payout{}, ErrForbidden
	}
	if !validID(submissionID) {
		return Payout{}, ErrInvalid
	}
	var p Payout
	var campaignStatus, submissionStatus string
	err := s.pool.QueryRow(ctx, `SELECT cs.id,c.id,c.name,cp.id,pr.display_name,cs.status,c.status,(floor(least(m.views_count::numeric*c.cpm_amount/1000,c.max_payout_per_clip)*100)/100)::numeric(14,2)::text,c.currency_code,m.views_count,m.captured_at FROM clip_submissions cs JOIN campaigns c ON c.id=cs.campaign_id JOIN campaign_participants cp ON cp.id=cs.participant_id JOIN profiles pr ON pr.user_id=cp.clipper_id JOIN LATERAL(SELECT * FROM metric_snapshots WHERE submission_id=cs.id ORDER BY captured_at DESC LIMIT 1)m ON true WHERE cs.id=$1 AND c.cpm_amount>0`, submissionID).Scan(&p.SubmissionID, &p.CampaignID, &p.CampaignTitle, &p.ParticipantID, &p.DisplayName, &submissionStatus, &campaignStatus, &p.Amount, &p.Currency, &p.BasisViews, &p.BasisAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrMetrics
	}
	if err != nil {
		return p, err
	}
	if campaignStatus != "COMPLETED" {
		return p, ErrCampaign
	}
	if submissionStatus != "APPROVED" || p.Amount == "0.00" {
		return p, ErrMetrics
	}
	p.Status = "QUOTE"
	return p, nil
}

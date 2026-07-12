package moderation

import "time"

const (
	StatusPending  = "PENDING"
	StatusApproved = "APPROVED"
	StatusRejected = "REJECTED"
	StatusFlagged  = "FLAGGED"
)

type Requirement struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Position    int    `json:"position"`
}

type Submission struct {
	ID                 string        `json:"id"`
	CampaignID         string        `json:"campaign_id"`
	CampaignTitle      string        `json:"campaign_title"`
	ParticipantID      string        `json:"participant_id"`
	ClipperID          string        `json:"clipper_id"`
	ClipperDisplayName string        `json:"clipper_display_name"`
	Platform           string        `json:"platform"`
	ContentURL         string        `json:"content_url"`
	Caption            string        `json:"caption"`
	Status             string        `json:"status"`
	Reason             *string       `json:"reason,omitempty"`
	ReviewedByUserID   *string       `json:"reviewed_by_user_id,omitempty"`
	SubmittedAt        time.Time     `json:"submitted_at"`
	ReviewedAt         *time.Time    `json:"reviewed_at,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
	Requirements       []Requirement `json:"requirements,omitempty"`
	BrandID            string        `json:"-"`
}

type QueueQuery struct {
	Page, Limit                int
	Status, Platform, Search   string
	CampaignID, BrandID        string
	SubmittedFrom, SubmittedTo string
	ReviewedFrom, ReviewedTo   string
	Sort, Direction            string
	UnresolvedOnly             bool
}

type Page struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type ActionInput struct {
	Reason    string
	RequestID string
	IPAddress string
	UserAgent string
}

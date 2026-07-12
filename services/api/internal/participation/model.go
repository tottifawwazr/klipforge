package participation

import "time"

type Participation struct {
	ID         string     `json:"id"`
	CampaignID string     `json:"campaign_id"`
	ClipperID  string     `json:"clipper_id"`
	Status     string     `json:"status"`
	JoinedAt   *time.Time `json:"joined_at"`
	CreatedAt  time.Time  `json:"created_at"`
}
type Submission struct {
	ID              string     `json:"id"`
	CampaignID      string     `json:"campaign_id"`
	ParticipantID   string     `json:"participant_id"`
	Platform        string     `json:"platform"`
	ContentURL      string     `json:"content_url"`
	Caption         string     `json:"caption"`
	Status          string     `json:"status"`
	SubmittedAt     time.Time  `json:"submitted_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`
}
type Page struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}
type ListQuery struct {
	Page, Limit                                   int
	CampaignID, Platform, Status, Sort, Direction string
}
type SubmitInput struct{ Platform, ContentURL, Caption, RequestID, IPAddress, UserAgent string }
type UpdateInput struct {
	Platform, ContentURL, Caption   *string
	RequestID, IPAddress, UserAgent string
}

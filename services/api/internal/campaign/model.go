package campaign

import "time"

const (
	StatusDraft     = "DRAFT"
	StatusActive    = "ACTIVE"
	StatusPaused    = "PAUSED"
	StatusCompleted = "COMPLETED"
	StatusCancelled = "CANCELLED"
)

type Requirement struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Position    int    `json:"position"`
}

type Campaign struct {
	ID                   string        `json:"id"`
	BrandID              string        `json:"brand_id"`
	Title                string        `json:"title"`
	Slug                 string        `json:"slug"`
	Description          string        `json:"description"`
	Brief                string        `json:"brief"`
	Status               string        `json:"status"`
	TotalBudget          string        `json:"total_budget"`
	RemainingBudget      string        `json:"remaining_budget"`
	CPMRate              *string       `json:"cpm_rate"`
	MaximumPayoutPerClip string        `json:"maximum_payout_per_clip"`
	CurrencyCode         string        `json:"currency_code"`
	StartDate            time.Time     `json:"start_date"`
	EndDate              time.Time     `json:"end_date"`
	ThumbnailURL         *string       `json:"thumbnail_url"`
	Platforms            []string      `json:"platforms"`
	Requirements         []Requirement `json:"requirements"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`
}

type PublicCampaign struct {
	ID           string        `json:"id"`
	Title        string        `json:"title"`
	Slug         string        `json:"slug"`
	Description  string        `json:"description"`
	Brief        string        `json:"brief"`
	Status       string        `json:"status"`
	StartDate    time.Time     `json:"start_date"`
	EndDate      time.Time     `json:"end_date"`
	ThumbnailURL *string       `json:"thumbnail_url"`
	Platforms    []string      `json:"platforms"`
	Requirements []Requirement `json:"requirements"`
}

func (c Campaign) Public() PublicCampaign {
	return PublicCampaign{c.ID, c.Title, c.Slug, c.Description, c.Brief, c.Status, c.StartDate, c.EndDate, c.ThumbnailURL, c.Platforms, c.Requirements}
}

type CreateInput struct {
	Title                string
	Slug                 string
	Description          string
	Brief                string
	TotalBudget          string
	CPMRate              string
	MaximumPayoutPerClip string
	StartDate            string
	EndDate              string
	ThumbnailURL         *string
	Platforms            []string
	Requirements         []Requirement
	RequestID            string
	IPAddress            string
	UserAgent            string
}

type UpdateInput struct {
	Title        *string
	Description  *string
	Brief        *string
	StartDate    *string
	EndDate      *string
	ThumbnailURL *string
	Platforms    *[]string
	Requirements *[]Requirement
	RequestID    string
	IPAddress    string
	UserAgent    string
}

type ListQuery struct {
	Page      int
	Limit     int
	Search    string
	Status    string
	Platform  string
	BrandID   string
	StartDate string
	EndDate   string
	Sort      string
	Direction string
}

type Page struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

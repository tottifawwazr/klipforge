package campaign

import (
	"context"
	"math/big"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/klipforge/klipforge/services/api/internal/auth"
)

var slugInvalid = regexp.MustCompile(`[^a-z0-9]+`)
var requirementType = regexp.MustCompile(`^[A-Z][A-Z_]{0,31}$`)

type Service struct {
	repository *Repository
	now        func() time.Time
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository, now: time.Now}
}

func (s *Service) Create(ctx context.Context, actor auth.Principal, input CreateInput) (Campaign, error) {
	if actor.Role != auth.RoleBrand {
		return Campaign{}, auth.ErrRoleNotAllowed
	}
	c := Campaign{BrandID: actor.UserID, Title: strings.TrimSpace(input.Title), Slug: normalizeSlug(input.Slug, input.Title), Description: strings.TrimSpace(input.Description), Brief: strings.TrimSpace(input.Brief), TotalBudget: strings.TrimSpace(input.TotalBudget), CPMRate: strPtr(strings.TrimSpace(input.CPMRate)), MaximumPayoutPerClip: strings.TrimSpace(input.MaximumPayoutPerClip), ThumbnailURL: normalizeURL(input.ThumbnailURL), Platforms: normalizePlatforms(input.Platforms), Requirements: normalizeRequirements(input.Requirements)}
	var err error
	c.StartDate, err = parseDate(input.StartDate)
	if err != nil {
		return Campaign{}, ErrInvalidDateRange
	}
	c.EndDate, err = parseDate(input.EndDate)
	if err != nil {
		return Campaign{}, ErrInvalidDateRange
	}
	if err = validateCampaign(c, false); err != nil {
		return Campaign{}, err
	}
	return s.repository.Create(ctx, c, actor, input.RequestID, input.IPAddress, input.UserAgent)
}

func (s *Service) PublicList(ctx context.Context, q ListQuery) ([]PublicCampaign, Page, error) {
	q = normalizeQuery(q)
	if q.Status != "" && q.Status != StatusActive {
		return nil, Page{}, ErrInvalidQuery
	}
	if !validListQuery(q, false) {
		return nil, Page{}, ErrInvalidQuery
	}
	q.Status = StatusActive
	q.BrandID = ""
	items, page, err := s.repository.List(ctx, q)
	if err != nil {
		return nil, Page{}, err
	}
	out := make([]PublicCampaign, len(items))
	for i := range items {
		out[i] = items[i].Public()
	}
	return out, page, nil
}
func (s *Service) BrandList(ctx context.Context, actor auth.Principal, q ListQuery) ([]Campaign, Page, error) {
	if actor.Role != auth.RoleBrand {
		return nil, Page{}, auth.ErrRoleNotAllowed
	}
	q = normalizeQuery(q)
	if !validListQuery(q, false) {
		return nil, Page{}, ErrInvalidQuery
	}
	q.BrandID = actor.UserID
	return s.repository.List(ctx, q)
}
func (s *Service) AdminList(ctx context.Context, actor auth.Principal, q ListQuery) ([]Campaign, Page, error) {
	if actor.Role != auth.RoleAdmin {
		return nil, Page{}, auth.ErrRoleNotAllowed
	}
	q = normalizeQuery(q)
	if !validListQuery(q, true) {
		return nil, Page{}, ErrInvalidQuery
	}
	return s.repository.List(ctx, q)
}
func (s *Service) PublicGet(ctx context.Context, id string) (PublicCampaign, error) {
	c, err := s.repository.Get(ctx, id)
	if err != nil {
		return PublicCampaign{}, err
	}
	if c.Status != StatusActive {
		return PublicCampaign{}, ErrNotFound
	}
	return c.Public(), nil
}
func (s *Service) BrandGet(ctx context.Context, actor auth.Principal, id string) (Campaign, error) {
	c, err := s.repository.Get(ctx, id)
	if err != nil {
		return Campaign{}, err
	}
	if actor.Role != auth.RoleBrand || c.BrandID != actor.UserID {
		return Campaign{}, ErrAccessDenied
	}
	return c, nil
}
func (s *Service) AdminGet(ctx context.Context, actor auth.Principal, id string) (Campaign, error) {
	if actor.Role != auth.RoleAdmin {
		return Campaign{}, auth.ErrRoleNotAllowed
	}
	return s.repository.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, actor auth.Principal, id string, input UpdateInput) (Campaign, error) {
	c, err := s.BrandGet(ctx, actor, id)
	if err != nil {
		return Campaign{}, err
	}
	if c.Status != StatusDraft && c.Status != StatusPaused {
		return Campaign{}, ErrInvalidTransition
	}
	if input.Title != nil {
		c.Title = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		c.Description = strings.TrimSpace(*input.Description)
	}
	if input.Brief != nil {
		c.Brief = strings.TrimSpace(*input.Brief)
	}
	if input.ThumbnailURL != nil {
		c.ThumbnailURL = normalizeURL(input.ThumbnailURL)
	}
	if input.StartDate != nil {
		c.StartDate, err = parseDate(*input.StartDate)
		if err != nil {
			return Campaign{}, ErrInvalidDateRange
		}
	}
	if input.EndDate != nil {
		c.EndDate, err = parseDate(*input.EndDate)
		if err != nil {
			return Campaign{}, ErrInvalidDateRange
		}
	}
	if err = validateCampaign(c, false); err != nil {
		return Campaign{}, err
	}
	var platforms *[]string
	if input.Platforms != nil {
		value := normalizePlatforms(*input.Platforms)
		if !validPlatforms(value) {
			return Campaign{}, ErrInvalidPlatform
		}
		platforms = &value
	}
	var requirements *[]Requirement
	if input.Requirements != nil {
		value := normalizeRequirements(*input.Requirements)
		if !validRequirements(value) {
			return Campaign{}, ErrInvalidQuery
		}
		requirements = &value
	}
	return s.repository.Update(ctx, c, platforms, requirements, actor, input.RequestID, input.IPAddress, input.UserAgent)
}

func (s *Service) Publish(ctx context.Context, actor auth.Principal, id, requestID, ip, userAgent string) (Campaign, error) {
	c, err := s.BrandGet(ctx, actor, id)
	if err != nil {
		return Campaign{}, err
	}
	if c.Status == StatusCompleted || c.Status == StatusCancelled {
		return Campaign{}, ErrAlreadyTerminal
	}
	if c.Status != StatusDraft {
		return Campaign{}, ErrInvalidTransition
	}
	if err = validateCampaign(c, true); err != nil {
		return Campaign{}, ErrNotReadyToPublish
	}
	if !c.EndDate.After(s.now().UTC().Truncate(24 * time.Hour)) {
		return Campaign{}, ErrNotReadyToPublish
	}
	return s.repository.Transition(ctx, id, StatusDraft, StatusActive, "campaign.published", actor, requestID, ip, userAgent)
}
func (s *Service) Pause(ctx context.Context, actor auth.Principal, id, requestID, ip, userAgent string) (Campaign, error) {
	return s.brandTransition(ctx, actor, id, StatusActive, StatusPaused, "campaign.paused", requestID, ip, userAgent)
}
func (s *Service) Resume(ctx context.Context, actor auth.Principal, id, requestID, ip, userAgent string) (Campaign, error) {
	c, err := s.BrandGet(ctx, actor, id)
	if err != nil {
		return Campaign{}, err
	}
	if c.Status == StatusCompleted || c.Status == StatusCancelled {
		return Campaign{}, ErrAlreadyTerminal
	}
	if c.Status != StatusPaused {
		return Campaign{}, ErrInvalidTransition
	}
	if err = validateCampaign(c, true); err != nil {
		return Campaign{}, ErrNotReadyToPublish
	}
	if !c.EndDate.After(s.now().UTC().Truncate(24*time.Hour)) || !positive(c.RemainingBudget) {
		return Campaign{}, ErrNotReadyToPublish
	}
	return s.repository.Transition(ctx, id, StatusPaused, StatusActive, "campaign.resumed", actor, requestID, ip, userAgent)
}
func (s *Service) Complete(ctx context.Context, actor auth.Principal, id, requestID, ip, userAgent string) (Campaign, error) {
	c, err := s.BrandGet(ctx, actor, id)
	if err != nil {
		return Campaign{}, err
	}
	if c.Status == StatusCompleted || c.Status == StatusCancelled {
		return Campaign{}, ErrAlreadyTerminal
	}
	if c.Status != StatusActive && c.Status != StatusPaused {
		return Campaign{}, ErrInvalidTransition
	}
	return s.repository.Transition(ctx, id, c.Status, StatusCompleted, "campaign.completed", actor, requestID, ip, userAgent)
}
func (s *Service) Cancel(ctx context.Context, actor auth.Principal, id, requestID, ip, userAgent string) (Campaign, error) {
	c, err := s.BrandGet(ctx, actor, id)
	if err != nil {
		return Campaign{}, err
	}
	return s.cancel(ctx, c, actor, "campaign.cancelled", requestID, ip, userAgent)
}
func (s *Service) AdminCancel(ctx context.Context, actor auth.Principal, id, requestID, ip, userAgent string) (Campaign, error) {
	if actor.Role != auth.RoleAdmin {
		return Campaign{}, auth.ErrRoleNotAllowed
	}
	c, err := s.repository.Get(ctx, id)
	if err != nil {
		return Campaign{}, err
	}
	return s.cancel(ctx, c, actor, "campaign.admin_cancelled", requestID, ip, userAgent)
}
func (s *Service) cancel(ctx context.Context, c Campaign, actor auth.Principal, action, requestID, ip, userAgent string) (Campaign, error) {
	if c.Status == StatusCompleted || c.Status == StatusCancelled {
		return Campaign{}, ErrAlreadyTerminal
	}
	if c.Status != StatusDraft && c.Status != StatusActive && c.Status != StatusPaused {
		return Campaign{}, ErrInvalidTransition
	}
	return s.repository.Transition(ctx, c.ID, c.Status, StatusCancelled, action, actor, requestID, ip, userAgent)
}
func (s *Service) brandTransition(ctx context.Context, actor auth.Principal, id, from, to, action, requestID, ip, userAgent string) (Campaign, error) {
	c, err := s.BrandGet(ctx, actor, id)
	if err != nil {
		return Campaign{}, err
	}
	if c.Status == StatusCompleted || c.Status == StatusCancelled {
		return Campaign{}, ErrAlreadyTerminal
	}
	if c.Status != from {
		return Campaign{}, ErrInvalidTransition
	}
	return s.repository.Transition(ctx, id, from, to, action, actor, requestID, ip, userAgent)
}

func normalizeQuery(q ListQuery) ListQuery {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	q.Search = strings.TrimSpace(q.Search)
	q.Platform = strings.ToUpper(strings.TrimSpace(q.Platform))
	q.Status = strings.ToUpper(strings.TrimSpace(q.Status))
	q.Sort = strings.ToLower(strings.TrimSpace(q.Sort))
	q.Direction = strings.ToLower(strings.TrimSpace(q.Direction))
	q.StartDate = strings.TrimSpace(q.StartDate)
	q.EndDate = strings.TrimSpace(q.EndDate)
	if q.Direction == "" {
		q.Direction = "desc"
	}
	return q
}
func validListQuery(q ListQuery, allowBrand bool) bool {
	if q.Status != "" && !validStatus(q.Status) {
		return false
	}
	if q.Platform != "" && !validPlatforms([]string{q.Platform}) {
		return false
	}
	if q.Sort != "" && q.Sort != "created_at" && q.Sort != "start_date" && q.Sort != "end_date" && q.Sort != "title" {
		return false
	}
	if q.Direction != "asc" && q.Direction != "desc" {
		return false
	}
	if q.BrandID != "" && !allowBrand {
		return false
	}
	if q.StartDate != "" {
		if _, err := parseDate(q.StartDate); err != nil {
			return false
		}
	}
	if q.EndDate != "" {
		if _, err := parseDate(q.EndDate); err != nil {
			return false
		}
	}
	return true
}
func validStatus(status string) bool {
	return status == StatusDraft || status == StatusActive || status == StatusPaused || status == StatusCompleted || status == StatusCancelled
}
func validateCampaign(c Campaign, publishing bool) error {
	if len(c.Title) < 3 || len(c.Title) > 120 {
		return ErrTitleRequired
	}
	if c.Slug == "" || len(c.Slug) > 140 {
		return ErrTitleRequired
	}
	if len(c.Description) < 10 || len(c.Description) > 5000 || len(c.Brief) < 10 || len(c.Brief) > 1000 {
		return ErrInvalidQuery
	}
	if !c.EndDate.After(c.StartDate) {
		return ErrInvalidDateRange
	}
	if !positive(c.TotalBudget) || c.CPMRate == nil || !positive(*c.CPMRate) || !nonNegative(c.MaximumPayoutPerClip) {
		return ErrInvalidBudget
	}
	if greater(c.MaximumPayoutPerClip, c.TotalBudget) {
		return ErrInvalidBudget
	}
	if c.ThumbnailURL != nil && !validURL(*c.ThumbnailURL) {
		return ErrInvalidQuery
	}
	if !validPlatforms(c.Platforms) {
		return ErrInvalidPlatform
	}
	if !validRequirements(c.Requirements) {
		return ErrInvalidQuery
	}
	if publishing && len(c.Platforms) == 0 {
		return ErrNotReadyToPublish
	}
	return nil
}
func validPlatforms(values []string) bool {
	seen := map[string]bool{}
	for _, v := range values {
		if v != "TIKTOK" && v != "INSTAGRAM" && v != "YOUTUBE" || seen[v] {
			return false
		}
		seen[v] = true
	}
	return true
}
func validRequirements(values []Requirement) bool {
	if len(values) > 50 {
		return false
	}
	for _, v := range values {
		if !requirementType.MatchString(v.Type) || len(v.Description) < 3 || len(v.Description) > 500 {
			return false
		}
	}
	return true
}
func normalizePlatforms(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, strings.ToUpper(strings.TrimSpace(v)))
	}
	return out
}
func normalizeRequirements(values []Requirement) []Requirement {
	out := make([]Requirement, 0, len(values))
	for _, v := range values {
		out = append(out, Requirement{Type: strings.ToUpper(strings.TrimSpace(v.Type)), Description: strings.TrimSpace(v.Description)})
	}
	return out
}
func normalizeSlug(slug, title string) string {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		slug = strings.ToLower(strings.TrimSpace(title))
	}
	slug = slugInvalid.ReplaceAllString(slug, "-")
	return strings.Trim(slug, "-")
}
func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}
func strPtr(value string) *string { return &value }
func normalizeURL(value *string) *string {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	return &v
}
func validURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}
func rat(value string) *big.Rat {
	v := new(big.Rat)
	if _, ok := v.SetString(value); !ok {
		return nil
	}
	return v
}
func positive(value string) bool    { v := rat(value); return v != nil && v.Sign() > 0 }
func nonNegative(value string) bool { v := rat(value); return v != nil && v.Sign() >= 0 }
func greater(a, b string) bool      { x, y := rat(a), rat(b); return x == nil || y == nil || x.Cmp(y) > 0 }

package participation

import (
	"context"
	"errors"
	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/klipforge/klipforge/services/api/internal/campaign"
	"net/url"
	"strings"
	"time"
)

type Service struct {
	repo      *Repository
	campaigns *campaign.Repository
	now       func() time.Time
}

func NewService(repo *Repository, campaigns *campaign.Repository) *Service {
	return &Service{repo, campaigns, time.Now}
}
func (s *Service) Join(ctx context.Context, a auth.Principal, campaignID, requestID, ip, agent string) (Participation, error) {
	if a.Role != auth.RoleClipper {
		return Participation{}, auth.ErrRoleNotAllowed
	}
	c, err := s.campaigns.Get(ctx, campaignID)
	if err != nil {
		return Participation{}, campaignError(err)
	}
	if err := s.validateEligibleCampaign(c); err != nil {
		return Participation{}, err
	}
	return s.repo.Join(ctx, campaignID, a, requestID, ip, agent)
}
func (s *Service) MyParticipation(ctx context.Context, a auth.Principal, campaignID string) (Participation, error) {
	if a.Role != auth.RoleClipper {
		return Participation{}, auth.ErrRoleNotAllowed
	}
	return s.repo.Participation(ctx, campaignID, a.UserID)
}
func (s *Service) MyCampaigns(ctx context.Context, a auth.Principal, q ListQuery) ([]Participation, Page, error) {
	if a.Role != auth.RoleClipper {
		return nil, Page{}, auth.ErrRoleNotAllowed
	}
	q = normalizeQuery(q)
	if !validParticipationQuery(q) {
		return nil, Page{}, ErrInvalidQuery
	}
	return s.repo.ListParticipations(ctx, a.UserID, q.CampaignID, "", q)
}
func (s *Service) BrandParticipants(ctx context.Context, a auth.Principal, campaignID string, q ListQuery) ([]Participation, Page, error) {
	c, err := s.campaigns.Get(ctx, campaignID)
	if err != nil {
		return nil, Page{}, campaignError(err)
	}
	if a.Role != auth.RoleBrand || c.BrandID != a.UserID {
		return nil, Page{}, ErrCampaignNotFound
	}
	q = normalizeQuery(q)
	if !validParticipationQuery(q) {
		return nil, Page{}, ErrInvalidQuery
	}
	return s.repo.ListParticipations(ctx, "", campaignID, a.UserID, q)
}
func (s *Service) Submit(ctx context.Context, a auth.Principal, campaignID string, in SubmitInput) (Submission, error) {
	if a.Role != auth.RoleClipper {
		return Submission{}, auth.ErrRoleNotAllowed
	}
	c, err := s.campaigns.Get(ctx, campaignID)
	if err != nil {
		return Submission{}, campaignError(err)
	}
	if err := s.validateEligibleCampaign(c); err != nil {
		return Submission{}, err
	}
	p, err := s.repo.Participation(ctx, campaignID, a.UserID)
	if err != nil {
		if errors.Is(err, ErrParticipationNotFound) {
			return Submission{}, ErrSubmissionRequiresParticipation
		}
		return Submission{}, err
	}
	if p.Status != "ACCEPTED" {
		return Submission{}, ErrSubmissionRequiresParticipation
	}
	platform := strings.ToUpper(strings.TrimSpace(in.Platform))
	if !validPlatform(platform) {
		return Submission{}, ErrInvalidPlatform
	}
	if !campaignAllowsPlatform(c, platform) {
		return Submission{}, ErrPlatformNotAllowed
	}
	normalized, err := normalizeURL(in.ContentURL, platform)
	if err != nil {
		return Submission{}, err
	}
	caption := strings.TrimSpace(in.Caption)
	if len(caption) > 2200 {
		return Submission{}, ErrInvalidQuery
	}
	return s.repo.CreateSubmission(ctx, Submission{CampaignID: campaignID, ParticipantID: p.ID, Platform: platform, ContentURL: normalized, Caption: caption}, a, in.RequestID, in.IPAddress, in.UserAgent)
}
func (s *Service) MySubmission(ctx context.Context, a auth.Principal, id string) (Submission, error) {
	sub, clipper, _, err := s.repo.Submission(ctx, id)
	if err != nil {
		return Submission{}, err
	}
	if a.Role != auth.RoleClipper || clipper != a.UserID {
		return Submission{}, ErrSubmissionNotFound
	}
	return sub, nil
}
func (s *Service) Update(ctx context.Context, a auth.Principal, id string, in UpdateInput) (Submission, error) {
	sub, clipper, _, err := s.repo.Submission(ctx, id)
	if err != nil {
		return Submission{}, err
	}
	if a.Role != auth.RoleClipper || clipper != a.UserID {
		return Submission{}, ErrSubmissionNotFound
	}
	if sub.Status != "PENDING" {
		return Submission{}, ErrNotEditable
	}
	if in.Platform == nil && in.ContentURL == nil && in.Caption == nil {
		return Submission{}, ErrInvalidQuery
	}
	if in.Platform != nil {
		sub.Platform = strings.ToUpper(strings.TrimSpace(*in.Platform))
		if !validPlatform(sub.Platform) {
			return Submission{}, ErrInvalidPlatform
		}
	}
	if in.Caption != nil {
		sub.Caption = strings.TrimSpace(*in.Caption)
		if len(sub.Caption) > 2200 {
			return Submission{}, ErrInvalidQuery
		}
	}
	c, e := s.campaigns.Get(ctx, sub.CampaignID)
	if e != nil {
		return Submission{}, campaignError(e)
	}
	if err := s.validateEligibleCampaign(c); err != nil {
		return Submission{}, err
	}
	if !campaignAllowsPlatform(c, sub.Platform) {
		return Submission{}, ErrPlatformNotAllowed
	}
	contentURL := sub.ContentURL
	if in.ContentURL != nil {
		contentURL = *in.ContentURL
	}
	normalized, e := normalizeURL(contentURL, sub.Platform)
	if e != nil {
		return Submission{}, e
	}
	sub.ContentURL = normalized
	return s.repo.UpdateSubmission(ctx, sub, a, in.RequestID, in.IPAddress, in.UserAgent)
}
func (s *Service) BrandSubmissions(ctx context.Context, a auth.Principal, campaignID string, q ListQuery) ([]Submission, Page, error) {
	c, err := s.campaigns.Get(ctx, campaignID)
	if err != nil {
		return nil, Page{}, campaignError(err)
	}
	if a.Role != auth.RoleBrand || c.BrandID != a.UserID {
		return nil, Page{}, ErrCampaignNotFound
	}
	q = normalizeQuery(q)
	if !validSubmissionQuery(q) {
		return nil, Page{}, ErrInvalidQuery
	}
	return s.repo.ListSubmissions(ctx, "", a.UserID, campaignID, q)
}
func (s *Service) BrandSubmission(ctx context.Context, a auth.Principal, campaignID, id string) (Submission, error) {
	sub, _, brand, err := s.repo.Submission(ctx, id)
	if err != nil {
		return Submission{}, err
	}
	if sub.CampaignID != campaignID {
		return Submission{}, ErrSubmissionNotFound
	}
	if a.Role != auth.RoleBrand || brand != a.UserID {
		return Submission{}, ErrSubmissionNotFound
	}
	return sub, nil
}
func (s *Service) MySubmissions(ctx context.Context, a auth.Principal, q ListQuery) ([]Submission, Page, error) {
	if a.Role != auth.RoleClipper {
		return nil, Page{}, auth.ErrRoleNotAllowed
	}
	q = normalizeQuery(q)
	if !validSubmissionQuery(q) {
		return nil, Page{}, ErrInvalidQuery
	}
	return s.repo.ListSubmissions(ctx, a.UserID, "", q.CampaignID, q)
}

func (s *Service) AdminParticipations(ctx context.Context, a auth.Principal, q ListQuery) ([]Participation, Page, error) {
	if a.Role != auth.RoleAdmin {
		return nil, Page{}, auth.ErrRoleNotAllowed
	}
	q = normalizeQuery(q)
	if !validParticipationQuery(q) {
		return nil, Page{}, ErrInvalidQuery
	}
	return s.repo.ListParticipations(ctx, "", q.CampaignID, "", q)
}

func (s *Service) AdminParticipation(ctx context.Context, a auth.Principal, id string) (Participation, error) {
	if a.Role != auth.RoleAdmin {
		return Participation{}, auth.ErrRoleNotAllowed
	}
	return s.repo.ParticipationByID(ctx, id)
}

func (s *Service) AdminSubmissions(ctx context.Context, a auth.Principal, q ListQuery) ([]Submission, Page, error) {
	if a.Role != auth.RoleAdmin {
		return nil, Page{}, auth.ErrRoleNotAllowed
	}
	q = normalizeQuery(q)
	if !validSubmissionQuery(q) {
		return nil, Page{}, ErrInvalidQuery
	}
	return s.repo.ListSubmissions(ctx, "", "", q.CampaignID, q)
}

func (s *Service) AdminSubmission(ctx context.Context, a auth.Principal, id string) (Submission, error) {
	if a.Role != auth.RoleAdmin {
		return Submission{}, auth.ErrRoleNotAllowed
	}
	submission, _, _, err := s.repo.Submission(ctx, id)
	return submission, err
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
	q.Platform = strings.ToUpper(strings.TrimSpace(q.Platform))
	q.Status = strings.ToUpper(strings.TrimSpace(q.Status))
	q.Sort = strings.ToLower(strings.TrimSpace(q.Sort))
	q.Direction = strings.ToLower(strings.TrimSpace(q.Direction))
	if q.Direction == "" {
		q.Direction = "desc"
	}
	return q
}

func validParticipationQuery(q ListQuery) bool {
	return (q.Status == "" || validParticipationStatus(q.Status)) &&
		(q.Sort == "" || q.Sort == "created_at" || q.Sort == "joined_at" || q.Sort == "status") &&
		(q.Direction == "asc" || q.Direction == "desc")
}

func validSubmissionQuery(q ListQuery) bool {
	return (q.Platform == "" || validPlatform(q.Platform)) &&
		(q.Status == "" || validSubmissionStatus(q.Status)) &&
		(q.Sort == "" || q.Sort == "submitted_at" || q.Sort == "created_at" || q.Sort == "updated_at" || q.Sort == "status") &&
		(q.Direction == "asc" || q.Direction == "desc")
}

func validPlatform(platform string) bool {
	return platform == "TIKTOK" || platform == "INSTAGRAM" || platform == "YOUTUBE"
}

func validParticipationStatus(status string) bool {
	return status == "PENDING" || status == "ACCEPTED" || status == "DECLINED" || status == "REMOVED"
}

func validSubmissionStatus(status string) bool {
	return status == "PENDING" || status == "APPROVED" || status == "REJECTED" || status == "FLAGGED"
}

func campaignAllowsPlatform(c campaign.Campaign, platform string) bool {
	for _, allowed := range c.Platforms {
		if allowed == platform {
			return true
		}
	}
	return false
}

func (s *Service) validateEligibleCampaign(c campaign.Campaign) error {
	if c.Status != campaign.StatusActive {
		return ErrCampaignNotActive
	}
	today := s.now().UTC().Truncate(24 * time.Hour)
	if c.StartDate.After(today) {
		return ErrCampaignNotStarted
	}
	if !c.EndDate.After(today) {
		return ErrCampaignEnded
	}
	return nil
}

func campaignError(err error) error {
	if errors.Is(err, campaign.ErrNotFound) {
		return ErrCampaignNotFound
	}
	return err
}
func normalizeURL(raw, platform string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Port() != "" {
		return "", ErrInvalidURL
	}
	host := strings.ToLower(u.Hostname())
	canonicalHost := strings.TrimPrefix(host, "www.")
	valid := false
	switch platform {
	case "TIKTOK":
		valid = canonicalHost == "tiktok.com" || strings.HasSuffix(canonicalHost, ".tiktok.com")
	case "INSTAGRAM":
		valid = canonicalHost == "instagram.com" || strings.HasSuffix(canonicalHost, ".instagram.com")
	case "YOUTUBE":
		valid = canonicalHost == "youtube.com" || canonicalHost == "youtu.be" || strings.HasSuffix(canonicalHost, ".youtube.com")
	}
	if !valid {
		return "", ErrInvalidURL
	}
	u.Host = canonicalHost
	u.User = nil
	u.RawQuery = ""
	u.ForceQuery = false
	u.Fragment = ""
	u.RawPath = ""
	u.Path = strings.TrimRight(u.Path, "/")
	if u.Path == "" {
		return "", ErrInvalidURL
	}
	return u.String(), nil
}

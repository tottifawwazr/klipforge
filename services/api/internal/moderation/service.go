package moderation

import (
	"context"
	"encoding/hex"
	"strings"
	"time"

	"github.com/klipforge/klipforge/services/api/internal/auth"
)

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) BrandQueue(ctx context.Context, actor auth.Principal, campaignID string, query QueueQuery) ([]Submission, Page, error) {
	if err := validateActor(actor, auth.RoleBrand); err != nil {
		return nil, Page{}, err
	}
	if !validID(campaignID) {
		return nil, Page{}, ErrCampaignNotFound
	}
	owner, err := s.repository.CampaignOwner(ctx, campaignID)
	if err != nil || owner != actor.UserID {
		if err != nil && err != ErrCampaignNotFound {
			return nil, Page{}, err
		}
		return nil, Page{}, ErrCampaignNotFound
	}
	query = normalizeQuery(query)
	if query.CampaignID != "" && query.CampaignID != campaignID {
		return nil, Page{}, ErrInvalidQuery
	}
	if !validQuery(query, false) {
		return nil, Page{}, ErrInvalidQuery
	}
	query.CampaignID = campaignID
	query.BrandID = actor.UserID
	query.UnresolvedOnly = query.Status == ""
	return s.repository.List(ctx, query)
}

func (s *Service) AdminQueue(ctx context.Context, actor auth.Principal, query QueueQuery) ([]Submission, Page, error) {
	if err := validateActor(actor, auth.RoleAdmin); err != nil {
		return nil, Page{}, err
	}
	query = normalizeQuery(query)
	if !validQuery(query, true) {
		return nil, Page{}, ErrInvalidQuery
	}
	return s.repository.List(ctx, query)
}

func (s *Service) BrandDetail(ctx context.Context, actor auth.Principal, campaignID, submissionID string) (Submission, error) {
	if err := validateActor(actor, auth.RoleBrand); err != nil {
		return Submission{}, err
	}
	if !validID(campaignID) || !validID(submissionID) {
		return Submission{}, ErrSubmissionNotFound
	}
	submission, err := s.repository.Get(ctx, submissionID)
	if err != nil {
		return Submission{}, err
	}
	if submission.CampaignID != campaignID || submission.BrandID != actor.UserID {
		return Submission{}, ErrSubmissionNotFound
	}
	return submission, nil
}

func (s *Service) AdminDetail(ctx context.Context, actor auth.Principal, submissionID string) (Submission, error) {
	if err := validateActor(actor, auth.RoleAdmin); err != nil {
		return Submission{}, err
	}
	if !validID(submissionID) {
		return Submission{}, ErrSubmissionNotFound
	}
	return s.repository.Get(ctx, submissionID)
}

func (s *Service) BrandApprove(ctx context.Context, actor auth.Principal, campaignID, submissionID string, input ActionInput) (Submission, error) {
	return s.decide(ctx, actor, auth.RoleBrand, campaignID, submissionID, StatusApproved, input)
}

func (s *Service) BrandReject(ctx context.Context, actor auth.Principal, campaignID, submissionID string, input ActionInput) (Submission, error) {
	return s.decide(ctx, actor, auth.RoleBrand, campaignID, submissionID, StatusRejected, input)
}

func (s *Service) BrandFlag(ctx context.Context, actor auth.Principal, campaignID, submissionID string, input ActionInput) (Submission, error) {
	return s.decide(ctx, actor, auth.RoleBrand, campaignID, submissionID, StatusFlagged, input)
}

func (s *Service) AdminApprove(ctx context.Context, actor auth.Principal, submissionID string, input ActionInput) (Submission, error) {
	return s.decide(ctx, actor, auth.RoleAdmin, "", submissionID, StatusApproved, input)
}

func (s *Service) AdminReject(ctx context.Context, actor auth.Principal, submissionID string, input ActionInput) (Submission, error) {
	return s.decide(ctx, actor, auth.RoleAdmin, "", submissionID, StatusRejected, input)
}

func (s *Service) AdminFlag(ctx context.Context, actor auth.Principal, submissionID string, input ActionInput) (Submission, error) {
	return s.decide(ctx, actor, auth.RoleAdmin, "", submissionID, StatusFlagged, input)
}

func (s *Service) decide(ctx context.Context, actor auth.Principal, role, campaignID, submissionID, target string, input ActionInput) (Submission, error) {
	if err := validateActor(actor, role); err != nil {
		return Submission{}, err
	}
	if !validID(submissionID) || campaignID != "" && !validID(campaignID) {
		return Submission{}, ErrSubmissionNotFound
	}
	reason := strings.TrimSpace(input.Reason)
	switch target {
	case StatusRejected:
		if reason == "" {
			return Submission{}, ErrRejectionReasonRequired
		}
		if len([]rune(reason)) > 1000 {
			return Submission{}, ErrRejectionReasonTooLong
		}
	case StatusFlagged:
		if reason == "" {
			return Submission{}, ErrFlagReasonRequired
		}
		if len([]rune(reason)) > 1000 {
			return Submission{}, ErrFlagReasonTooLong
		}
	case StatusApproved:
		reason = ""
	default:
		return Submission{}, ErrInvalidTransition
	}
	input.Reason = reason
	return s.repository.Moderate(ctx, actor, campaignID, submissionID, target, input)
}

func normalizeQuery(query QueueQuery) QueueQuery {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit < 1 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	query.Status = strings.ToUpper(strings.TrimSpace(query.Status))
	query.Platform = strings.ToUpper(strings.TrimSpace(query.Platform))
	query.Search = strings.TrimSpace(query.Search)
	query.CampaignID = strings.TrimSpace(query.CampaignID)
	query.BrandID = strings.TrimSpace(query.BrandID)
	query.SubmittedFrom = strings.TrimSpace(query.SubmittedFrom)
	query.SubmittedTo = strings.TrimSpace(query.SubmittedTo)
	query.ReviewedFrom = strings.TrimSpace(query.ReviewedFrom)
	query.ReviewedTo = strings.TrimSpace(query.ReviewedTo)
	query.Sort = strings.ToLower(strings.TrimSpace(query.Sort))
	query.Direction = strings.ToLower(strings.TrimSpace(query.Direction))
	if query.Sort == "" {
		query.Sort = "submitted_at"
	}
	if query.Direction == "" {
		query.Direction = "asc"
	}
	return query
}

func validQuery(query QueueQuery, admin bool) bool {
	if len([]rune(query.Search)) > 200 {
		return false
	}
	if query.Status != "" && !validStatus(query.Status) || query.Platform != "" && !validPlatform(query.Platform) {
		return false
	}
	if query.Sort != "submitted_at" && query.Sort != "reviewed_at" && query.Sort != "created_at" && query.Sort != "status" && query.Sort != "platform" {
		return false
	}
	if query.Direction != "asc" && query.Direction != "desc" {
		return false
	}
	if query.CampaignID != "" && !validID(query.CampaignID) || query.BrandID != "" && (!admin || !validID(query.BrandID)) {
		return false
	}
	for _, value := range []string{query.SubmittedFrom, query.SubmittedTo, query.ReviewedFrom, query.ReviewedTo} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return false
			}
		}
	}
	return true
}

func validateActor(actor auth.Principal, role string) error {
	if !actor.AccountActive {
		return auth.ErrUserInactive
	}
	if !actor.SessionActive {
		return auth.ErrSessionRevoked
	}
	if actor.Role != role {
		return auth.ErrRoleNotAllowed
	}
	return nil
}

func validStatus(status string) bool {
	return status == StatusPending || status == StatusApproved || status == StatusRejected || status == StatusFlagged
}

func validPlatform(platform string) bool {
	return platform == "TIKTOK" || platform == "INSTAGRAM" || platform == "YOUTUBE"
}

func validID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	_, err := hex.DecodeString(strings.ReplaceAll(value, "-", ""))
	return err == nil
}

func validTransition(from, to string) bool {
	if from == StatusPending {
		return to == StatusApproved || to == StatusRejected || to == StatusFlagged
	}
	if from == StatusFlagged {
		return to == StatusApproved || to == StatusRejected
	}
	return false
}

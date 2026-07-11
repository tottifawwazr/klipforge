package authorization

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/klipforge/klipforge/services/api/internal/auth"
)

type Policies struct{ repository *Repository }

func NewPolicies(repository *Repository) *Policies { return &Policies{repository: repository} }

func (p *Policies) CanManageCampaign(ctx context.Context, principal auth.Principal, campaignID string) error {
	if err := validatePrincipal(principal); err != nil {
		return err
	}
	record, err := p.campaign(ctx, campaignID)
	if err != nil {
		return err
	}
	if principal.Role == auth.RoleAdmin {
		return nil
	}
	if principal.Role == auth.RoleBrand && record.BrandID == principal.UserID {
		return nil
	}
	if principal.Role == auth.RoleBrand {
		return auth.ErrResourceNotFound
	}
	return auth.ErrRoleNotAllowed
}

func (p *Policies) CanViewCampaign(ctx context.Context, principal auth.Principal, campaignID string) error {
	if err := validatePrincipal(principal); err != nil {
		return err
	}
	record, err := p.campaign(ctx, campaignID)
	if err != nil {
		return err
	}
	if principal.Role == auth.RoleAdmin || (principal.Role == auth.RoleBrand && record.BrandID == principal.UserID) {
		return nil
	}
	if principal.Role == auth.RoleClipper && record.Status == "ACTIVE" {
		return nil
	}
	return auth.ErrResourceNotFound
}

func (p *Policies) CanJoinCampaign(ctx context.Context, principal auth.Principal, campaignID string) error {
	if err := validatePrincipal(principal); err != nil {
		return err
	}
	if principal.Role != auth.RoleClipper {
		return auth.ErrRoleNotAllowed
	}
	record, err := p.campaign(ctx, campaignID)
	if err != nil {
		return err
	}
	if record.Status != "ACTIVE" {
		return auth.ErrResourceNotFound
	}
	return nil
}

func (p *Policies) CanViewParticipation(ctx context.Context, principal auth.Principal, participationID string) error {
	if err := validatePrincipal(principal); err != nil {
		return err
	}
	record, err := p.participation(ctx, participationID)
	if err != nil {
		return err
	}
	if principal.Role == auth.RoleAdmin || (principal.Role == auth.RoleClipper && record.ClipperID == principal.UserID) ||
		(principal.Role == auth.RoleBrand && record.BrandID == principal.UserID) {
		return nil
	}
	return auth.ErrResourceNotFound
}

func (p *Policies) CanManageSubmission(ctx context.Context, principal auth.Principal, submissionID string) error {
	if err := validatePrincipal(principal); err != nil {
		return err
	}
	record, err := p.submission(ctx, submissionID)
	if err != nil {
		return err
	}
	if principal.Role != auth.RoleClipper {
		return auth.ErrRoleNotAllowed
	}
	if record.ClipperID != principal.UserID {
		return auth.ErrResourceNotFound
	}
	if record.Status != "PENDING" && record.Status != "FLAGGED" {
		return auth.ErrForbidden
	}
	return nil
}

func (p *Policies) CanReviewSubmission(ctx context.Context, principal auth.Principal, submissionID string) error {
	if err := validatePrincipal(principal); err != nil {
		return err
	}
	record, err := p.submission(ctx, submissionID)
	if err != nil {
		return err
	}
	if principal.Role == auth.RoleAdmin || (principal.Role == auth.RoleBrand && record.BrandID == principal.UserID) {
		return nil
	}
	if principal.Role == auth.RoleBrand {
		return auth.ErrResourceNotFound
	}
	return auth.ErrRoleNotAllowed
}

func (p *Policies) CanModerateSubmission(ctx context.Context, principal auth.Principal, submissionID string) error {
	if err := validatePrincipal(principal); err != nil {
		return err
	}
	if principal.Role != auth.RoleAdmin {
		return auth.ErrRoleNotAllowed
	}
	_, err := p.submission(ctx, submissionID)
	return err
}

func (p *Policies) CanViewPayout(ctx context.Context, principal auth.Principal, payoutID string) error {
	if err := validatePrincipal(principal); err != nil {
		return err
	}
	record, err := p.payout(ctx, payoutID)
	if err != nil {
		return err
	}
	if principal.Role == auth.RoleAdmin || (principal.Role == auth.RoleClipper && record.ClipperID == principal.UserID) ||
		(principal.Role == auth.RoleBrand && record.BrandID == principal.UserID) {
		return nil
	}
	return auth.ErrResourceNotFound
}

func (p *Policies) CanProcessPayout(ctx context.Context, principal auth.Principal, payoutID string) error {
	if err := validatePrincipal(principal); err != nil {
		return err
	}
	if principal.Role != auth.RoleAdmin {
		return auth.ErrRoleNotAllowed
	}
	_, err := p.payout(ctx, payoutID)
	return err
}

func (p *Policies) CanViewPrivateProfile(ctx context.Context, principal auth.Principal, userID string) error {
	if err := validatePrincipal(principal); err != nil {
		return err
	}
	if err := p.profile(ctx, userID); err != nil {
		return err
	}
	if principal.UserID == userID || principal.Role == auth.RoleAdmin {
		return nil
	}
	return auth.ErrResourceNotFound
}

func (p *Policies) CanUpdateProfile(ctx context.Context, principal auth.Principal, userID string) error {
	if err := validatePrincipal(principal); err != nil {
		return err
	}
	if err := p.profile(ctx, userID); err != nil {
		return err
	}
	if principal.UserID == userID {
		return nil
	}
	return auth.ErrResourceNotFound
}

func (p *Policies) campaign(ctx context.Context, id string) (campaignRecord, error) {
	if !validResourceID(id) {
		return campaignRecord{}, auth.ErrResourceNotFound
	}
	record, err := p.repository.campaign(ctx, id)
	return record, mapResourceError(err)
}

func (p *Policies) participation(ctx context.Context, id string) (participationRecord, error) {
	if !validResourceID(id) {
		return participationRecord{}, auth.ErrResourceNotFound
	}
	record, err := p.repository.participation(ctx, id)
	return record, mapResourceError(err)
}

func (p *Policies) submission(ctx context.Context, id string) (submissionRecord, error) {
	if !validResourceID(id) {
		return submissionRecord{}, auth.ErrResourceNotFound
	}
	record, err := p.repository.submission(ctx, id)
	return record, mapResourceError(err)
}

func (p *Policies) payout(ctx context.Context, id string) (payoutRecord, error) {
	if !validResourceID(id) {
		return payoutRecord{}, auth.ErrResourceNotFound
	}
	record, err := p.repository.payout(ctx, id)
	return record, mapResourceError(err)
}

func (p *Policies) profile(ctx context.Context, id string) error {
	if !validResourceID(id) {
		return auth.ErrResourceNotFound
	}
	return mapResourceError(p.repository.profileExists(ctx, id))
}

func validatePrincipal(principal auth.Principal) error {
	if !auth.ValidRole(principal.Role) {
		return auth.ErrRoleNotAllowed
	}
	if !principal.AccountActive {
		return auth.ErrUserInactive
	}
	if !principal.SessionActive {
		return auth.ErrSessionRevoked
	}
	return nil
}

func mapResourceError(err error) error {
	if errors.Is(err, errResourceMissing) {
		return auth.ErrResourceNotFound
	}
	return err
}

func validResourceID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	_, err := hex.DecodeString(strings.ReplaceAll(value, "-", ""))
	return err == nil
}

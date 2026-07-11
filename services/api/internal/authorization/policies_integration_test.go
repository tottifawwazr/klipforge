package authorization

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/klipforge/klipforge/services/api/internal/auth"
)

const (
	adminID            = "00000000-0000-0000-0000-000000000001"
	brandOneID         = "00000000-0000-0000-0000-000000000101"
	brandTwoID         = "00000000-0000-0000-0000-000000000102"
	clipperOneID       = "00000000-0000-0000-0000-000000000201"
	clipperTwoID       = "00000000-0000-0000-0000-000000000202"
	campaignOneID      = "00000000-0000-0000-0000-000000000301"
	privateCampaignID  = "00000000-0000-0000-0000-000000000303"
	inactiveCampaignID = "00000000-0000-0000-0000-000000000306"
	participationOneID = "00000000-0000-0000-0000-000000000501"
	submissionOneID    = "00000000-0000-0000-0000-000000000601"
	submissionTwoID    = "00000000-0000-0000-0000-000000000602"
	payoutOneID        = "00000000-0000-0000-0000-000000000901"
	missingID          = "ffffffff-ffff-4fff-8fff-ffffffffffff"
)

func integrationPolicies(t *testing.T) (*Policies, context.Context) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	var seeded bool
	if err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM campaigns WHERE id=$1)`, campaignOneID).Scan(&seeded); err != nil || !seeded {
		t.Fatalf("authorization fixtures are not seeded: seeded=%v err=%v", seeded, err)
	}
	return NewPolicies(NewRepository(pool)), ctx
}

func principal(userID, role string) auth.Principal {
	return auth.Principal{UserID: userID, Role: role, SessionID: "00000000-0000-0000-0000-00000000f001", TokenID: "00000000-0000-0000-0000-00000000f002", AccountActive: true, SessionActive: true}
}

func TestCampaignPolicies(t *testing.T) {
	p, ctx := integrationPolicies(t)
	tests := []struct {
		name      string
		actor     auth.Principal
		operation func(context.Context, auth.Principal, string) error
		resource  string
		want      error
	}{
		{"brand owner manages", principal(brandOneID, auth.RoleBrand), p.CanManageCampaign, campaignOneID, nil},
		{"other brand hidden", principal(brandTwoID, auth.RoleBrand), p.CanManageCampaign, campaignOneID, auth.ErrResourceNotFound},
		{"admin explicit management", principal(adminID, auth.RoleAdmin), p.CanManageCampaign, campaignOneID, nil},
		{"clipper cannot manage", principal(clipperOneID, auth.RoleClipper), p.CanManageCampaign, campaignOneID, auth.ErrRoleNotAllowed},
		{"clipper views active", principal(clipperOneID, auth.RoleClipper), p.CanViewCampaign, campaignOneID, nil},
		{"clipper cannot view private", principal(clipperOneID, auth.RoleClipper), p.CanViewCampaign, privateCampaignID, auth.ErrResourceNotFound},
		{"clipper cannot view inactive", principal(clipperOneID, auth.RoleClipper), p.CanViewCampaign, inactiveCampaignID, auth.ErrResourceNotFound},
		{"missing campaign", principal(adminID, auth.RoleAdmin), p.CanManageCampaign, missingID, auth.ErrResourceNotFound},
		{"malformed campaign ID", principal(adminID, auth.RoleAdmin), p.CanManageCampaign, "not-a-uuid", auth.ErrResourceNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.operation(ctx, test.actor, test.resource); got != test.want {
				t.Fatalf("error=%v want=%v", got, test.want)
			}
		})
	}
}

func TestParticipationPolicies(t *testing.T) {
	p, ctx := integrationPolicies(t)
	if err := p.CanJoinCampaign(ctx, principal(clipperOneID, auth.RoleClipper), campaignOneID); err != nil {
		t.Fatalf("clipper join: %v", err)
	}
	for _, actor := range []auth.Principal{principal(brandOneID, auth.RoleBrand), principal(adminID, auth.RoleAdmin)} {
		if err := p.CanJoinCampaign(ctx, actor, campaignOneID); err != auth.ErrRoleNotAllowed {
			t.Fatalf("role %s join error=%v", actor.Role, err)
		}
	}
	for _, actor := range []auth.Principal{principal(clipperOneID, auth.RoleClipper), principal(brandOneID, auth.RoleBrand), principal(adminID, auth.RoleAdmin)} {
		if err := p.CanViewParticipation(ctx, actor, participationOneID); err != nil {
			t.Fatalf("role %s view participation: %v", actor.Role, err)
		}
	}
	if err := p.CanViewParticipation(ctx, principal(clipperTwoID, auth.RoleClipper), participationOneID); err != auth.ErrResourceNotFound {
		t.Fatalf("other clipper participation error=%v", err)
	}
}

func TestSubmissionPoliciesAndHorizontalEscalation(t *testing.T) {
	p, ctx := integrationPolicies(t)
	if err := p.CanManageSubmission(ctx, principal(clipperTwoID, auth.RoleClipper), submissionTwoID); err != nil {
		t.Fatalf("own pending submission: %v", err)
	}
	if err := p.CanManageSubmission(ctx, principal(clipperOneID, auth.RoleClipper), submissionTwoID); err != auth.ErrResourceNotFound {
		t.Fatalf("other clipper submission error=%v", err)
	}
	if err := p.CanReviewSubmission(ctx, principal(brandOneID, auth.RoleBrand), submissionOneID); err != nil {
		t.Fatalf("campaign owner review: %v", err)
	}
	if err := p.CanReviewSubmission(ctx, principal(brandTwoID, auth.RoleBrand), submissionOneID); err != auth.ErrResourceNotFound {
		t.Fatalf("unrelated brand review error=%v", err)
	}
	if err := p.CanModerateSubmission(ctx, principal(adminID, auth.RoleAdmin), submissionOneID); err != nil {
		t.Fatalf("admin moderation: %v", err)
	}
	if err := p.CanModerateSubmission(ctx, principal(brandOneID, auth.RoleBrand), submissionOneID); err != auth.ErrRoleNotAllowed {
		t.Fatalf("brand moderation error=%v", err)
	}
}

func TestPayoutPoliciesAndHorizontalEscalation(t *testing.T) {
	p, ctx := integrationPolicies(t)
	if err := p.CanViewPayout(ctx, principal(clipperOneID, auth.RoleClipper), payoutOneID); err != nil {
		t.Fatalf("own payout: %v", err)
	}
	if err := p.CanViewPayout(ctx, principal(clipperTwoID, auth.RoleClipper), payoutOneID); err != auth.ErrResourceNotFound {
		t.Fatalf("other clipper payout error=%v", err)
	}
	if err := p.CanViewPayout(ctx, principal(brandOneID, auth.RoleBrand), payoutOneID); err != nil {
		t.Fatalf("campaign owner payout: %v", err)
	}
	if err := p.CanViewPayout(ctx, principal(brandTwoID, auth.RoleBrand), payoutOneID); err != auth.ErrResourceNotFound {
		t.Fatalf("other brand obligation error=%v", err)
	}
	if err := p.CanProcessPayout(ctx, principal(adminID, auth.RoleAdmin), payoutOneID); err != nil {
		t.Fatalf("admin process policy: %v", err)
	}
	for _, actor := range []auth.Principal{principal(clipperOneID, auth.RoleClipper), principal(brandOneID, auth.RoleBrand)} {
		if err := p.CanProcessPayout(ctx, actor, payoutOneID); err != auth.ErrRoleNotAllowed {
			t.Fatalf("role %s process error=%v", actor.Role, err)
		}
	}
}

func TestProfilePoliciesAndHorizontalEscalation(t *testing.T) {
	p, ctx := integrationPolicies(t)
	clipper := principal(clipperOneID, auth.RoleClipper)
	if err := p.CanViewPrivateProfile(ctx, clipper, clipperOneID); err != nil {
		t.Fatal(err)
	}
	if err := p.CanUpdateProfile(ctx, clipper, clipperOneID); err != nil {
		t.Fatal(err)
	}
	if err := p.CanViewPrivateProfile(ctx, clipper, clipperTwoID); err != auth.ErrResourceNotFound {
		t.Fatalf("private profile escalation=%v", err)
	}
	if err := p.CanUpdateProfile(ctx, principal(adminID, auth.RoleAdmin), clipperOneID); err != auth.ErrResourceNotFound {
		t.Fatalf("admin update bypass must be explicit and absent: %v", err)
	}
	if err := p.CanViewPrivateProfile(ctx, principal(adminID, auth.RoleAdmin), clipperOneID); err != nil {
		t.Fatalf("admin view: %v", err)
	}
}

func TestPoliciesEnforcePrincipalState(t *testing.T) {
	p, ctx := integrationPolicies(t)
	inactive := principal(brandOneID, auth.RoleBrand)
	inactive.AccountActive = false
	if err := p.CanManageCampaign(ctx, inactive, campaignOneID); err != auth.ErrUserInactive {
		t.Fatalf("inactive error=%v", err)
	}
	revoked := principal(brandOneID, auth.RoleBrand)
	revoked.SessionActive = false
	if err := p.CanManageCampaign(ctx, revoked, campaignOneID); err != auth.ErrSessionRevoked {
		t.Fatalf("revoked error=%v", err)
	}
	invalid := principal(brandOneID, "OWNER")
	if err := p.CanManageCampaign(ctx, invalid, campaignOneID); err != auth.ErrRoleNotAllowed {
		t.Fatalf("invalid role error=%v", err)
	}
}

package moderation

import (
	"strings"
	"testing"

	"github.com/klipforge/klipforge/services/api/internal/auth"
)

func TestModerationTransitions(t *testing.T) {
	tests := []struct {
		from, to string
		allowed  bool
	}{
		{StatusPending, StatusApproved, true},
		{StatusPending, StatusRejected, true},
		{StatusPending, StatusFlagged, true},
		{StatusFlagged, StatusApproved, true},
		{StatusFlagged, StatusRejected, true},
		{StatusFlagged, StatusFlagged, false},
		{StatusApproved, StatusRejected, false},
		{StatusRejected, StatusApproved, false},
		{StatusApproved, StatusPending, false},
		{StatusRejected, StatusPending, false},
	}
	for _, test := range tests {
		if got := validTransition(test.from, test.to); got != test.allowed {
			t.Fatalf("transition %s -> %s = %v, want %v", test.from, test.to, got, test.allowed)
		}
	}
}

func TestModerationReasonValidation(t *testing.T) {
	service := NewService(nil)
	brand := auth.Principal{UserID: "00000000-0000-0000-0000-000000000101", Role: auth.RoleBrand, AccountActive: true, SessionActive: true}
	campaignID := "00000000-0000-0000-0000-000000000301"
	submissionID := "00000000-0000-0000-0000-000000000601"

	if _, err := service.BrandReject(t.Context(), brand, campaignID, submissionID, ActionInput{Reason: "   "}); err != ErrRejectionReasonRequired {
		t.Fatalf("blank rejection reason error = %v", err)
	}
	if _, err := service.BrandReject(t.Context(), brand, campaignID, submissionID, ActionInput{Reason: strings.Repeat("x", 1001)}); err != ErrRejectionReasonTooLong {
		t.Fatalf("long rejection reason error = %v", err)
	}
	if _, err := service.BrandFlag(t.Context(), brand, campaignID, submissionID, ActionInput{Reason: "\t"}); err != ErrFlagReasonRequired {
		t.Fatalf("blank flag reason error = %v", err)
	}
	if _, err := service.BrandFlag(t.Context(), brand, campaignID, submissionID, ActionInput{Reason: strings.Repeat("x", 1001)}); err != ErrFlagReasonTooLong {
		t.Fatalf("long flag reason error = %v", err)
	}
}

func TestModerationQueryValidation(t *testing.T) {
	valid := normalizeQuery(QueueQuery{Status: "flagged", Platform: "instagram", Sort: "reviewed_at", Direction: "desc", SubmittedFrom: "2026-01-01"})
	if !validQuery(valid, false) {
		t.Fatal("valid moderation query rejected")
	}
	if validQuery(normalizeQuery(QueueQuery{Status: "UNKNOWN"}), false) {
		t.Fatal("unknown status accepted")
	}
	if validQuery(normalizeQuery(QueueQuery{Sort: "content_url"}), false) {
		t.Fatal("unsafe sort accepted")
	}
	if validQuery(normalizeQuery(QueueQuery{SubmittedFrom: "01-01-2026"}), false) {
		t.Fatal("invalid date accepted")
	}
	if validQuery(normalizeQuery(QueueQuery{Search: strings.Repeat("x", 201)}), false) {
		t.Fatal("oversized search accepted")
	}
	if validQuery(normalizeQuery(QueueQuery{BrandID: "00000000-0000-0000-0000-000000000101"}), false) {
		t.Fatal("brand filter accepted on brand queue")
	}
}

func TestModerationActorState(t *testing.T) {
	actor := auth.Principal{Role: auth.RoleAdmin, AccountActive: false, SessionActive: true}
	if err := validateActor(actor, auth.RoleAdmin); err != auth.ErrUserInactive {
		t.Fatalf("inactive actor error = %v", err)
	}
	actor.AccountActive = true
	actor.SessionActive = false
	if err := validateActor(actor, auth.RoleAdmin); err != auth.ErrSessionRevoked {
		t.Fatalf("revoked actor error = %v", err)
	}
	actor.SessionActive = true
	actor.Role = auth.RoleClipper
	if err := validateActor(actor, auth.RoleAdmin); err != auth.ErrRoleNotAllowed {
		t.Fatalf("role error = %v", err)
	}
}

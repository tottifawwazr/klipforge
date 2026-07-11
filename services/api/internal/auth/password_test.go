package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordPolicyAndBcrypt(t *testing.T) {
	for _, weak := range []string{"short1A", "lowercase1", "UPPERCASE1", "NoNumbers"} {
		if ValidatePassword(weak) {
			t.Fatalf("weak password accepted: %q", weak)
		}
	}
	if ValidatePassword(strings.Repeat("Aa1", 25)) {
		t.Fatal("password longer than bcrypt's 72-byte limit was accepted")
	}
	service := NewPasswordService(bcrypt.MinCost)
	hash, err := service.Hash("StrongPass1")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "StrongPass1" || !strings.HasPrefix(hash, "$2") {
		t.Fatalf("not a bcrypt hash: %q", hash)
	}
	if !service.Verify(hash, "StrongPass1") || service.Verify(hash, "WrongPass1") {
		t.Fatal("password verification mismatch")
	}
}

func TestEmailNormalizationAndValidation(t *testing.T) {
	if got := NormalizeEmail("  Person@Example.COM "); got != "person@example.com" {
		t.Fatalf("normalized email = %q", got)
	}
	if !ValidateEmail("person@example.com") || ValidateEmail("not-an-email") {
		t.Fatal("email validation mismatch")
	}
}

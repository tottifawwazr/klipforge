package auth

import "testing"

func TestValidRole(t *testing.T) {
	for _, role := range []string{RoleAdmin, RoleBrand, RoleClipper} {
		if !ValidRole(role) {
			t.Fatalf("valid role rejected: %s", role)
		}
	}
	for _, role := range []string{"", "admin", "OWNER", "BRAND "} {
		if ValidRole(role) {
			t.Fatalf("invalid role accepted: %q", role)
		}
	}
}

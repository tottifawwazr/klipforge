package auth

const (
	RoleAdmin   = "ADMIN"
	RoleBrand   = "BRAND"
	RoleClipper = "CLIPPER"
)

func ValidRole(role string) bool {
	switch role {
	case RoleAdmin, RoleBrand, RoleClipper:
		return true
	default:
		return false
	}
}

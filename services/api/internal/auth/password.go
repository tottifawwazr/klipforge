package auth

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type PasswordService struct{ cost int }

func NewPasswordService(cost int) PasswordService { return PasswordService{cost: cost} }

func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func ValidateEmail(email string) bool {
	return len(email) <= 254 && emailPattern.MatchString(email)
}

func ValidatePassword(password string) bool {
	// bcrypt only considers inputs up to 72 bytes; reject longer values instead
	// of silently truncating or surfacing a hashing error.
	if len([]rune(password)) < 8 || len(password) > 72 {
		return false
	}
	var upper, lower, number bool
	for _, r := range password {
		upper = upper || unicode.IsUpper(r)
		lower = lower || unicode.IsLower(r)
		number = number || unicode.IsNumber(r)
	}
	return upper && lower && number
}

func (s PasswordService) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	return string(hash), err
}

func (s PasswordService) Verify(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

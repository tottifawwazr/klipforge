package auth

import "errors"

type Error struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *Error) Error() string { return e.Code }

var (
	ErrValidation         = &Error{"VALIDATION_ERROR", "The request is invalid.", 400}
	ErrEmailRegistered    = &Error{"EMAIL_ALREADY_REGISTERED", "An account with this email already exists.", 409}
	ErrAuthentication     = &Error{"AUTHENTICATION_FAILED", "Authentication failed.", 401}
	ErrAccessMissing      = &Error{"ACCESS_TOKEN_MISSING", "An access token is required.", 401}
	ErrAccessInvalid      = &Error{"ACCESS_TOKEN_INVALID", "The access token is invalid.", 401}
	ErrAccessExpired      = &Error{"ACCESS_TOKEN_EXPIRED", "The access token has expired.", 401}
	ErrRefreshMissing     = &Error{"REFRESH_TOKEN_MISSING", "A refresh token is required.", 401}
	ErrRefreshInvalid     = &Error{"REFRESH_TOKEN_INVALID", "The refresh token is invalid.", 401}
	ErrRefreshExpired     = &Error{"REFRESH_TOKEN_EXPIRED", "The refresh token has expired.", 401}
	ErrRefreshReused      = &Error{"REFRESH_TOKEN_REUSED", "Refresh token reuse was detected. The session was revoked.", 401}
	ErrSessionRevoked     = &Error{"SESSION_REVOKED", "The session has been revoked.", 401}
	ErrUserInactive       = &Error{"USER_INACTIVE", "The user account is inactive.", 403}
	ErrRateLimited        = &Error{"RATE_LIMIT_EXCEEDED", "Too many authentication attempts. Try again later.", 429}
	ErrServiceUnavailable = &Error{"AUTH_SERVICE_UNAVAILABLE", "Authentication is temporarily unavailable.", 503}
)

func AsError(err error) *Error {
	var domain *Error
	if errors.As(err, &domain) {
		return domain
	}
	return nil
}

package campaign

import "errors"

type Error struct {
	Code, Message string
	HTTPStatus    int
}

func (e *Error) Error() string { return e.Code }

var (
	ErrNotFound          = &Error{"CAMPAIGN_NOT_FOUND", "The campaign was not found.", 404}
	ErrAccessDenied      = &Error{"CAMPAIGN_ACCESS_DENIED", "You do not have permission to access this campaign.", 404}
	ErrTitleRequired     = &Error{"CAMPAIGN_TITLE_REQUIRED", "A campaign title is required.", 400}
	ErrSlugConflict      = &Error{"CAMPAIGN_SLUG_CONFLICT", "That campaign slug is already in use.", 409}
	ErrInvalidPlatform   = &Error{"CAMPAIGN_INVALID_PLATFORM", "One or more campaign platforms are invalid.", 400}
	ErrInvalidDateRange  = &Error{"CAMPAIGN_INVALID_DATE_RANGE", "The campaign date range is invalid.", 400}
	ErrInvalidBudget     = &Error{"CAMPAIGN_INVALID_BUDGET", "The campaign budget values are invalid.", 400}
	ErrNotReadyToPublish = &Error{"CAMPAIGN_NOT_READY_TO_PUBLISH", "The campaign is not ready to publish.", 400}
	ErrInvalidTransition = &Error{"CAMPAIGN_INVALID_TRANSITION", "The requested campaign lifecycle transition is not allowed.", 409}
	ErrAlreadyTerminal   = &Error{"CAMPAIGN_ALREADY_TERMINAL", "The campaign is already in a terminal state.", 409}
	ErrInvalidQuery      = &Error{"VALIDATION_ERROR", "The campaign query is invalid.", 400}
)

func AsError(err error) *Error {
	var target *Error
	if errors.As(err, &target) {
		return target
	}
	return nil
}

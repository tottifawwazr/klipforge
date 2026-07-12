package moderation

import "errors"

type Error struct {
	Code, Message string
	HTTPStatus    int
}

func (e *Error) Error() string { return e.Code }

var (
	ErrCampaignNotFound        = &Error{"CAMPAIGN_NOT_FOUND", "The campaign was not found.", 404}
	ErrSubmissionNotFound      = &Error{"SUBMISSION_NOT_FOUND", "The submission was not found.", 404}
	ErrInvalidTransition       = &Error{"SUBMISSION_INVALID_TRANSITION", "The submission cannot make that moderation transition.", 409}
	ErrAlreadyReviewed         = &Error{"SUBMISSION_ALREADY_REVIEWED", "The submission has already reached a terminal review decision.", 409}
	ErrRejectionReasonRequired = &Error{"SUBMISSION_REJECTION_REASON_REQUIRED", "A rejection reason is required.", 400}
	ErrRejectionReasonTooLong  = &Error{"SUBMISSION_REJECTION_REASON_TOO_LONG", "The rejection reason must not exceed 1000 characters.", 400}
	ErrFlagReasonRequired      = &Error{"SUBMISSION_FLAG_REASON_REQUIRED", "A flag reason is required.", 400}
	ErrFlagReasonTooLong       = &Error{"SUBMISSION_FLAG_REASON_TOO_LONG", "The flag reason must not exceed 1000 characters.", 400}
	ErrModerationConflict      = &Error{"MODERATION_CONFLICT", "Another moderator changed this submission first.", 409}
	ErrSubmissionNotReady      = &Error{"SUBMISSION_NOT_MODERATION_READY", "The submission is not ready for moderation.", 409}
	ErrInvalidQuery            = &Error{"VALIDATION_ERROR", "The moderation request is invalid.", 400}
)

func AsError(err error) *Error {
	var target *Error
	if errors.As(err, &target) {
		return target
	}
	return nil
}

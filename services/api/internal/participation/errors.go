package participation

import "errors"

type Error struct {
	Code, Message string
	HTTPStatus    int
}

func (e *Error) Error() string { return e.Code }

var (
	ErrCampaignNotFound                = &Error{"CAMPAIGN_NOT_FOUND", "The campaign was not found.", 404}
	ErrCampaignNotActive               = &Error{"CAMPAIGN_NOT_ACTIVE", "The campaign is not active.", 409}
	ErrCampaignNotStarted              = &Error{"CAMPAIGN_NOT_STARTED", "The campaign has not started.", 409}
	ErrCampaignEnded                   = &Error{"CAMPAIGN_ENDED", "The campaign has ended.", 409}
	ErrAlreadyJoined                   = &Error{"CAMPAIGN_ALREADY_JOINED", "You have already joined this campaign.", 409}
	ErrParticipationNotFound           = &Error{"CAMPAIGN_PARTICIPATION_NOT_FOUND", "The participation record was not found.", 404}
	ErrSubmissionNotFound              = &Error{"SUBMISSION_NOT_FOUND", "The submission was not found.", 404}
	ErrSubmissionRequiresParticipation = &Error{"SUBMISSION_REQUIRES_PARTICIPATION", "Campaign participation is required.", 409}
	ErrDuplicateURL                    = &Error{"SUBMISSION_DUPLICATE_URL", "This content URL has already been submitted.", 409}
	ErrInvalidURL                      = &Error{"SUBMISSION_INVALID_URL", "The submission URL is invalid.", 400}
	ErrInvalidPlatform                 = &Error{"SUBMISSION_INVALID_PLATFORM", "The submission platform is invalid.", 400}
	ErrPlatformNotAllowed              = &Error{"SUBMISSION_PLATFORM_NOT_ALLOWED", "The platform is not enabled for this campaign.", 409}
	ErrNotEditable                     = &Error{"SUBMISSION_NOT_EDITABLE", "This submission can no longer be edited.", 409}
	ErrInvalidQuery                    = &Error{"VALIDATION_ERROR", "The request is invalid.", 400}
)

func AsError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

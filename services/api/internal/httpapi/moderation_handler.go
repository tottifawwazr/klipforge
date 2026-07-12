package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/klipforge/klipforge/services/api/internal/moderation"
)

type ModerationHandler struct {
	service       *moderation.Service
	authorization *AuthorizationMiddleware
	logger        *slog.Logger
}

func NewModerationHandler(service *moderation.Service, authorization *AuthorizationMiddleware, logger *slog.Logger) *ModerationHandler {
	return &ModerationHandler{service: service, authorization: authorization, logger: logger}
}

func (h *ModerationHandler) RegisterBrandRoutes(router chi.Router) {
	router.Get("/{campaignID}/moderation/submissions", h.brandQueue)
	router.Get("/{campaignID}/moderation/submissions/{submissionID}", h.brandDetail)
	router.Post("/{campaignID}/submissions/{submissionID}/approve", h.brandApprove)
	router.Post("/{campaignID}/submissions/{submissionID}/reject", h.brandReject)
	router.Post("/{campaignID}/submissions/{submissionID}/flag", h.brandFlag)
}

func (h *ModerationHandler) RegisterAdminRoutes(router chi.Router) {
	router.Get("/moderation/submissions", h.adminQueue)
	router.Get("/moderation/submissions/{submissionID}", h.adminDetail)
	router.Post("/submissions/{submissionID}/approve", h.adminApprove)
	router.Post("/submissions/{submissionID}/reject", h.adminReject)
	router.Post("/submissions/{submissionID}/flag", h.adminFlag)
}

func (h *ModerationHandler) brandQueue(w http.ResponseWriter, r *http.Request) {
	items, page, err := h.service.BrandQueue(r.Context(), principal(r), chi.URLParam(r, "campaignID"), moderationQuery(r))
	if err != nil {
		h.writeError(w, r, "moderation.brand_queue", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"submissions": items, "pagination": page})
}

func (h *ModerationHandler) brandDetail(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.BrandDetail(r.Context(), principal(r), chi.URLParam(r, "campaignID"), chi.URLParam(r, "submissionID"))
	if err != nil {
		h.writeError(w, r, "moderation.brand_detail", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"submission": item})
}

func (h *ModerationHandler) adminQueue(w http.ResponseWriter, r *http.Request) {
	items, page, err := h.service.AdminQueue(r.Context(), principal(r), moderationQuery(r))
	if err != nil {
		h.writeError(w, r, "moderation.admin_queue", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"submissions": items, "pagination": page})
}

func (h *ModerationHandler) adminDetail(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.AdminDetail(r.Context(), principal(r), chi.URLParam(r, "submissionID"))
	if err != nil {
		h.writeError(w, r, "moderation.admin_detail", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"submission": item})
}

func (h *ModerationHandler) brandApprove(w http.ResponseWriter, r *http.Request) {
	if !decodeOptionalEmptyJSON(r) {
		h.writeError(w, r, "submission.approve", moderation.ErrInvalidQuery)
		return
	}
	h.runAction(w, r, "submission.approve", func(input moderation.ActionInput) (moderation.Submission, error) {
		return h.service.BrandApprove(r.Context(), principal(r), chi.URLParam(r, "campaignID"), chi.URLParam(r, "submissionID"), input)
	})
}

func (h *ModerationHandler) brandReject(w http.ResponseWriter, r *http.Request) {
	h.runReasonAction(w, r, "submission.reject", func(input moderation.ActionInput) (moderation.Submission, error) {
		return h.service.BrandReject(r.Context(), principal(r), chi.URLParam(r, "campaignID"), chi.URLParam(r, "submissionID"), input)
	})
}

func (h *ModerationHandler) brandFlag(w http.ResponseWriter, r *http.Request) {
	h.runReasonAction(w, r, "submission.flag", func(input moderation.ActionInput) (moderation.Submission, error) {
		return h.service.BrandFlag(r.Context(), principal(r), chi.URLParam(r, "campaignID"), chi.URLParam(r, "submissionID"), input)
	})
}

func (h *ModerationHandler) adminApprove(w http.ResponseWriter, r *http.Request) {
	if !decodeOptionalEmptyJSON(r) {
		h.writeError(w, r, "submission.admin_approve", moderation.ErrInvalidQuery)
		return
	}
	h.runAction(w, r, "submission.admin_approve", func(input moderation.ActionInput) (moderation.Submission, error) {
		return h.service.AdminApprove(r.Context(), principal(r), chi.URLParam(r, "submissionID"), input)
	})
}

func (h *ModerationHandler) adminReject(w http.ResponseWriter, r *http.Request) {
	h.runReasonAction(w, r, "submission.admin_reject", func(input moderation.ActionInput) (moderation.Submission, error) {
		return h.service.AdminReject(r.Context(), principal(r), chi.URLParam(r, "submissionID"), input)
	})
}

func (h *ModerationHandler) adminFlag(w http.ResponseWriter, r *http.Request) {
	h.runReasonAction(w, r, "submission.admin_flag", func(input moderation.ActionInput) (moderation.Submission, error) {
		return h.service.AdminFlag(r.Context(), principal(r), chi.URLParam(r, "submissionID"), input)
	})
}

type moderationAction func(moderation.ActionInput) (moderation.Submission, error)

func (h *ModerationHandler) runReasonAction(w http.ResponseWriter, r *http.Request, action string, next moderationAction) {
	var body struct {
		Reason string `json:"reason"`
	}
	if !decodeJSON(r, &body) {
		h.writeError(w, r, action, moderation.ErrInvalidQuery)
		return
	}
	h.runActionWithReason(w, r, action, body.Reason, next)
}

func (h *ModerationHandler) runAction(w http.ResponseWriter, r *http.Request, action string, next moderationAction) {
	h.runActionWithReason(w, r, action, "", next)
}

func (h *ModerationHandler) runActionWithReason(w http.ResponseWriter, r *http.Request, action, reason string, next moderationAction) {
	started := time.Now()
	actor := principal(r)
	input := moderation.ActionInput{Reason: reason, RequestID: requestIDFromContext(r.Context()), IPAddress: remoteIP(r.RemoteAddr), UserAgent: r.UserAgent()}
	item, err := next(input)
	if err != nil {
		h.writeError(w, r, action, err)
		return
	}
	h.logger.InfoContext(r.Context(), "moderation operation completed",
		"request_id", input.RequestID, "actor_user_id", actor.UserID, "actor_role", actor.Role,
		"campaign_id", item.CampaignID, "submission_id", item.ID, "action", action,
		"result", "success", "duration_ms", time.Since(started).Milliseconds(),
	)
	writeJSON(w, http.StatusOK, map[string]any{"submission": item})
}

func (h *ModerationHandler) writeError(w http.ResponseWriter, r *http.Request, action string, err error) {
	if domain := moderation.AsError(err); domain != nil {
		h.logger.WarnContext(r.Context(), "moderation operation rejected",
			"request_id", requestIDFromContext(r.Context()), "action", action,
			"submission_id", chi.URLParam(r, "submissionID"), "campaign_id", chi.URLParam(r, "campaignID"),
			"result", "rejected", "error_code", domain.Code,
		)
		writeAPIError(w, r, domain.HTTPStatus, domain.Code, domain.Message)
		return
	}
	if domain := auth.AsError(err); domain != nil {
		writeAuthError(w, r, domain)
		return
	}
	h.logger.ErrorContext(r.Context(), "moderation operation failed", "request_id", requestIDFromContext(r.Context()), "action", action, "error", err)
	writeAPIError(w, r, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred.")
}

func moderationQuery(r *http.Request) moderation.QueueQuery {
	query := r.URL.Query()
	return moderation.QueueQuery{
		Page: parseInt(query.Get("page")), Limit: parseInt(query.Get("limit")),
		Status: strings.ToUpper(query.Get("status")), Platform: strings.ToUpper(query.Get("platform")), Search: query.Get("search"),
		CampaignID: query.Get("campaign_id"), BrandID: query.Get("brand_id"),
		SubmittedFrom: query.Get("submitted_from"), SubmittedTo: query.Get("submitted_to"),
		ReviewedFrom: query.Get("reviewed_from"), ReviewedTo: query.Get("reviewed_to"),
		Sort: query.Get("sort"), Direction: query.Get("direction"),
	}
}

func decodeOptionalEmptyJSON(r *http.Request) bool {
	if r.Body == nil || r.Body == http.NoBody {
		return true
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var body struct{}
	if err := decoder.Decode(&body); errors.Is(err, io.EOF) {
		return true
	} else if err != nil {
		return false
	}
	var extra any
	return errors.Is(decoder.Decode(&extra), io.EOF)
}

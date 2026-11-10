package httpapi

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/klipforge/klipforge/services/api/internal/operations"
	"log/slog"
	"net/http"
	"strconv"
)

type OperationsHandler struct {
	service *operations.Service
	logger  *slog.Logger
}

func NewOperationsHandler(service *operations.Service, logger *slog.Logger) *OperationsHandler {
	return &OperationsHandler{service, logger}
}
func (h *OperationsHandler) list(w http.ResponseWriter, r *http.Request) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			h.fail(w, r, operations.ErrInvalid)
			return
		}
		page = n
	}
	items, err := h.service.List(r.Context(), principal(r), page)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"payouts": items, "page": page, "page_size": 50, "payment_mode": "MANUAL_RECORD"})
}
func (h *OperationsHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SubmissionID   string `json:"submission_id"`
		ExpectedAmount string `json:"expected_amount"`
	}
	if !decodeJSON(r, &body) || body.ExpectedAmount == "" {
		h.fail(w, r, operations.ErrInvalid)
		return
	}
	p, replayed, err := h.service.Create(r.Context(), principal(r), body.SubmissionID, r.Header.Get("Idempotency-Key"), body.ExpectedAmount)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	code := 201
	if replayed {
		code = 200
	}
	writeJSON(w, code, map[string]any{"payout": p, "replayed": replayed, "payment_mode": "MANUAL_RECORD"})
}
func (h *OperationsHandler) transition(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Reference    string `json:"reference"`
		Reason       string `json:"reason"`
		Acknowledged bool   `json:"acknowledge_manual_record"`
	}
	if !decodeJSON(r, &body) {
		h.fail(w, r, operations.ErrInvalid)
		return
	}
	action := chi.URLParam(r, "action")
	if action == "process" && !body.Acknowledged {
		h.fail(w, r, operations.ErrInvalid)
		return
	}
	p, err := h.service.Transition(r.Context(), principal(r), chi.URLParam(r, "payoutID"), action, body.Reference, body.Reason)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"payout": p, "payment_mode": "MANUAL_RECORD"})
}
func (h *OperationsHandler) notifications(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Notifications(r.Context(), principal(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"notifications": items})
}
func (h *OperationsHandler) read(w http.ResponseWriter, r *http.Request) {
	if err := h.service.ReadNotification(r.Context(), principal(r), chi.URLParam(r, "notificationID")); err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"read": true})
}
func (h *OperationsHandler) audit(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.Audit(r.Context(), principal(r))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"events": items})
}
func (h *OperationsHandler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var domain *operations.Error
	if errors.As(err, &domain) {
		writeAPIError(w, r, domain.Status, domain.Code, domain.Message)
		return
	}
	h.logger.ErrorContext(r.Context(), "operations request failed", "request_id", requestIDFromContext(r.Context()), "error", err)
	writeAPIError(w, r, 500, "OPERATIONS_UNAVAILABLE", "This operation is temporarily unavailable.")
}

func (h *OperationsHandler) quote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SubmissionID string `json:"submission_id"`
	}
	if !decodeJSON(r, &body) {
		h.fail(w, r, operations.ErrInvalid)
		return
	}
	p, err := h.service.Quote(r.Context(), principal(r), body.SubmissionID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"quote": p, "payment_mode": "MANUAL_RECORD"})
}

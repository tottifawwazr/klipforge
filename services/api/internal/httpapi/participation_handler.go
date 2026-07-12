package httpapi

import (
	"github.com/go-chi/chi/v5"
	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/klipforge/klipforge/services/api/internal/participation"
	"net/http"
	"strings"
)

type ParticipationHandler struct {
	service       *participation.Service
	authorization *AuthorizationMiddleware
}

func NewParticipationHandler(s *participation.Service, a *AuthorizationMiddleware) *ParticipationHandler {
	return &ParticipationHandler{s, a}
}
func (h *ParticipationHandler) CampaignRoutes() http.Handler {
	r := chi.NewRouter()
	h.RegisterCampaignRoutes(r)
	return r
}
func (h *ParticipationHandler) RegisterCampaignRoutes(r chi.Router) {
	r.With(h.authorization.RequireAuthentication, h.authorization.RequireActiveUser, h.authorization.RequireActiveSession, h.authorization.RequireRole(auth.RoleClipper)).Post("/{campaignID}/join", h.join)
	r.With(h.authorization.RequireAuthentication, h.authorization.RequireActiveUser, h.authorization.RequireActiveSession, h.authorization.RequireRole(auth.RoleClipper)).Post("/{campaignID}/submissions", h.submit)
}
func (h *ParticipationHandler) ClipperRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(h.authorization.RequireAuthentication, h.authorization.RequireActiveUser, h.authorization.RequireActiveSession, h.authorization.RequireRole(auth.RoleClipper))
	r.Get("/campaigns", h.myCampaigns)
	r.Get("/campaigns/{campaignID}/participation", h.myParticipation)
	r.Get("/submissions", h.mySubmissions)
	r.Get("/submissions/{submissionID}", h.mySubmission)
	r.Patch("/submissions/{submissionID}", h.updateSubmission)
	return r
}
func (h *ParticipationHandler) BrandRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(h.authorization.RequireAuthentication, h.authorization.RequireActiveUser, h.authorization.RequireActiveSession, h.authorization.RequireRole(auth.RoleBrand))
	h.RegisterBrandRoutes(r)
	return r
}

func (h *ParticipationHandler) AdminRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(h.authorization.RequireAuthentication, h.authorization.RequireActiveUser, h.authorization.RequireActiveSession, h.authorization.RequireRole(auth.RoleAdmin))
	h.RegisterAdminRoutes(r)
	return r
}
func (h *ParticipationHandler) RegisterAdminRoutes(r chi.Router) {
	r.Get("/participations", h.adminParticipations)
	r.Get("/participations/{participationID}", h.adminParticipation)
	r.Get("/submissions", h.adminSubmissions)
	r.Get("/submissions/{submissionID}", h.adminSubmission)
}
func (h *ParticipationHandler) RegisterBrandRoutes(r chi.Router) {
	r.Get("/{campaignID}/participants", h.brandParticipants)
	r.Get("/{campaignID}/submissions", h.brandSubmissions)
	r.Get("/{campaignID}/submissions/{submissionID}", h.brandSubmission)
}
func (h *ParticipationHandler) join(w http.ResponseWriter, r *http.Request) {
	p := principal(r)
	result, err := h.service.Join(r.Context(), p, chi.URLParam(r, "campaignID"), requestIDFromContext(r.Context()), remoteIP(r.RemoteAddr), r.UserAgent())
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"participation": result})
}

type submissionRequest struct {
	Platform   string `json:"platform"`
	ContentURL string `json:"content_url"`
	Caption    string `json:"caption"`
}
type submissionPatch struct {
	Platform   *string `json:"platform"`
	ContentURL *string `json:"content_url"`
	Caption    *string `json:"caption"`
}

func (h *ParticipationHandler) submit(w http.ResponseWriter, r *http.Request) {
	var b submissionRequest
	if !decodeJSON(r, &b) {
		writeParticipationError(w, r, participation.ErrInvalidQuery)
		return
	}
	s, err := h.service.Submit(r.Context(), principal(r), chi.URLParam(r, "campaignID"), participation.SubmitInput{Platform: b.Platform, ContentURL: b.ContentURL, Caption: b.Caption, RequestID: requestIDFromContext(r.Context()), IPAddress: remoteIP(r.RemoteAddr), UserAgent: r.UserAgent()})
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"submission": s})
}
func (h *ParticipationHandler) myCampaigns(w http.ResponseWriter, r *http.Request) {
	items, page, err := h.service.MyCampaigns(r.Context(), principal(r), participationQuery(r))
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"participations": items, "pagination": page})
}
func (h *ParticipationHandler) myParticipation(w http.ResponseWriter, r *http.Request) {
	p, err := h.service.MyParticipation(r.Context(), principal(r), chi.URLParam(r, "campaignID"))
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"participation": p})
}
func (h *ParticipationHandler) mySubmissions(w http.ResponseWriter, r *http.Request) {
	items, page, err := h.service.MySubmissions(r.Context(), principal(r), participationQuery(r))
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"submissions": items, "pagination": page})
}
func (h *ParticipationHandler) mySubmission(w http.ResponseWriter, r *http.Request) {
	s, err := h.service.MySubmission(r.Context(), principal(r), chi.URLParam(r, "submissionID"))
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"submission": s})
}
func (h *ParticipationHandler) updateSubmission(w http.ResponseWriter, r *http.Request) {
	var b submissionPatch
	if !decodeJSON(r, &b) {
		writeParticipationError(w, r, participation.ErrInvalidQuery)
		return
	}
	s, err := h.service.Update(r.Context(), principal(r), chi.URLParam(r, "submissionID"), participation.UpdateInput{Platform: b.Platform, ContentURL: b.ContentURL, Caption: b.Caption, RequestID: requestIDFromContext(r.Context()), IPAddress: remoteIP(r.RemoteAddr), UserAgent: r.UserAgent()})
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"submission": s})
}
func (h *ParticipationHandler) brandParticipants(w http.ResponseWriter, r *http.Request) {
	items, page, err := h.service.BrandParticipants(r.Context(), principal(r), chi.URLParam(r, "campaignID"), participationQuery(r))
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"participants": items, "pagination": page})
}
func (h *ParticipationHandler) brandSubmissions(w http.ResponseWriter, r *http.Request) {
	items, page, err := h.service.BrandSubmissions(r.Context(), principal(r), chi.URLParam(r, "campaignID"), participationQuery(r))
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"submissions": items, "pagination": page})
}
func (h *ParticipationHandler) brandSubmission(w http.ResponseWriter, r *http.Request) {
	s, err := h.service.BrandSubmission(r.Context(), principal(r), chi.URLParam(r, "campaignID"), chi.URLParam(r, "submissionID"))
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"submission": s})
}

func (h *ParticipationHandler) adminParticipations(w http.ResponseWriter, r *http.Request) {
	items, page, err := h.service.AdminParticipations(r.Context(), principal(r), participationQuery(r))
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"participations": items, "pagination": page})
}

func (h *ParticipationHandler) adminParticipation(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.AdminParticipation(r.Context(), principal(r), chi.URLParam(r, "participationID"))
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"participation": item})
}

func (h *ParticipationHandler) adminSubmissions(w http.ResponseWriter, r *http.Request) {
	items, page, err := h.service.AdminSubmissions(r.Context(), principal(r), participationQuery(r))
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"submissions": items, "pagination": page})
}

func (h *ParticipationHandler) adminSubmission(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.AdminSubmission(r.Context(), principal(r), chi.URLParam(r, "submissionID"))
	if err != nil {
		writeParticipationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"submission": item})
}
func participationQuery(r *http.Request) participation.ListQuery {
	q := r.URL.Query()
	return participation.ListQuery{Page: parseInt(q.Get("page")), Limit: parseInt(q.Get("limit")), CampaignID: q.Get("campaign_id"), Platform: strings.ToUpper(q.Get("platform")), Status: strings.ToUpper(q.Get("status")), Sort: q.Get("sort"), Direction: q.Get("direction")}
}
func writeParticipationError(w http.ResponseWriter, r *http.Request, err error) {
	if e := participation.AsError(err); e != nil {
		writeAPIError(w, r, e.HTTPStatus, e.Code, e.Message)
		return
	}
	if e := auth.AsError(err); e != nil {
		writeAuthError(w, r, e)
		return
	}
	writeAPIError(w, r, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred.")
}

package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/klipforge/klipforge/services/api/internal/campaign"
)

type CampaignHandler struct {
	service       *campaign.Service
	authorization *AuthorizationMiddleware
	logger        *slog.Logger
}

func NewCampaignHandler(service *campaign.Service, authorization *AuthorizationMiddleware, logger *slog.Logger) *CampaignHandler {
	return &CampaignHandler{service, authorization, logger}
}

func (h *CampaignHandler) PublicRoutes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.publicList)
	r.Get("/{campaignID}", h.publicGet)
	r.With(h.authorization.RequireAuthentication, h.authorization.RequireActiveUser, h.authorization.RequireActiveSession, h.authorization.RequireRole(auth.RoleBrand)).Post("/", h.create)
	return r
}
func (h *CampaignHandler) BrandRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(h.authorization.RequireAuthentication, h.authorization.RequireActiveUser, h.authorization.RequireActiveSession, h.authorization.RequireRole(auth.RoleBrand))
	r.Get("/", h.brandList)
	r.Get("/{campaignID}", h.brandGet)
	r.Patch("/{campaignID}", h.update)
	r.Post("/{campaignID}/publish", h.publish)
	r.Post("/{campaignID}/pause", h.pause)
	r.Post("/{campaignID}/resume", h.resume)
	r.Post("/{campaignID}/complete", h.complete)
	r.Post("/{campaignID}/cancel", h.cancel)
	return r
}
func (h *CampaignHandler) AdminRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(h.authorization.RequireAuthentication, h.authorization.RequireActiveUser, h.authorization.RequireActiveSession, h.authorization.RequireRole(auth.RoleAdmin))
	r.Get("/", h.adminList)
	r.Get("/{campaignID}", h.adminGet)
	r.Post("/{campaignID}/cancel", h.adminCancel)
	return r
}

type campaignRequest struct {
	Title                string                 `json:"title"`
	Slug                 string                 `json:"slug"`
	Description          string                 `json:"description"`
	Brief                string                 `json:"brief"`
	TotalBudget          string                 `json:"total_budget"`
	CPMRate              string                 `json:"cpm_rate"`
	MaximumPayoutPerClip string                 `json:"maximum_payout_per_clip"`
	StartDate            string                 `json:"start_date"`
	EndDate              string                 `json:"end_date"`
	ThumbnailURL         *string                `json:"thumbnail_url"`
	Platforms            []string               `json:"platforms"`
	Requirements         []campaign.Requirement `json:"requirements"`
}
type campaignPatchRequest struct {
	Title        *string                 `json:"title"`
	Description  *string                 `json:"description"`
	Brief        *string                 `json:"brief"`
	StartDate    *string                 `json:"start_date"`
	EndDate      *string                 `json:"end_date"`
	ThumbnailURL *string                 `json:"thumbnail_url"`
	Platforms    *[]string               `json:"platforms"`
	Requirements *[]campaign.Requirement `json:"requirements"`
}

func (h *CampaignHandler) create(w http.ResponseWriter, r *http.Request) {
	var body campaignRequest
	if !decodeJSON(r, &body) {
		writeCampaignError(w, r, campaign.ErrInvalidQuery)
		return
	}
	p := principal(r)
	c, err := h.service.Create(r.Context(), p, campaign.CreateInput{Title: body.Title, Slug: body.Slug, Description: body.Description, Brief: body.Brief, TotalBudget: body.TotalBudget, CPMRate: body.CPMRate, MaximumPayoutPerClip: body.MaximumPayoutPerClip, StartDate: body.StartDate, EndDate: body.EndDate, ThumbnailURL: body.ThumbnailURL, Platforms: body.Platforms, Requirements: body.Requirements, RequestID: requestIDFromContext(r.Context()), IPAddress: remoteIP(r.RemoteAddr), UserAgent: r.UserAgent()})
	if err != nil {
		h.handleCampaignError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"campaign": c})
}
func (h *CampaignHandler) publicList(w http.ResponseWriter, r *http.Request) {
	items, page, err := h.service.PublicList(r.Context(), listQuery(r))
	if err != nil {
		h.handleCampaignError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaigns": items, "pagination": page})
}
func (h *CampaignHandler) publicGet(w http.ResponseWriter, r *http.Request) {
	c, err := h.service.PublicGet(r.Context(), chi.URLParam(r, "campaignID"))
	if err != nil {
		h.handleCampaignError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaign": c})
}
func (h *CampaignHandler) brandList(w http.ResponseWriter, r *http.Request) {
	items, page, err := h.service.BrandList(r.Context(), principal(r), listQuery(r))
	if err != nil {
		h.handleCampaignError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaigns": items, "pagination": page})
}
func (h *CampaignHandler) brandGet(w http.ResponseWriter, r *http.Request) {
	c, err := h.service.BrandGet(r.Context(), principal(r), chi.URLParam(r, "campaignID"))
	if err != nil {
		h.handleCampaignError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaign": c})
}
func (h *CampaignHandler) update(w http.ResponseWriter, r *http.Request) {
	var body campaignPatchRequest
	if !decodeJSON(r, &body) {
		writeCampaignError(w, r, campaign.ErrInvalidQuery)
		return
	}
	c, err := h.service.Update(r.Context(), principal(r), chi.URLParam(r, "campaignID"), campaign.UpdateInput{Title: body.Title, Description: body.Description, Brief: body.Brief, StartDate: body.StartDate, EndDate: body.EndDate, ThumbnailURL: body.ThumbnailURL, Platforms: body.Platforms, Requirements: body.Requirements, RequestID: requestIDFromContext(r.Context()), IPAddress: remoteIP(r.RemoteAddr), UserAgent: r.UserAgent()})
	if err != nil {
		h.handleCampaignError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaign": c})
}
func (h *CampaignHandler) publish(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, "publish", h.service.Publish)
}
func (h *CampaignHandler) pause(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, "pause", h.service.Pause)
}
func (h *CampaignHandler) resume(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, "resume", h.service.Resume)
}
func (h *CampaignHandler) complete(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, "complete", h.service.Complete)
}
func (h *CampaignHandler) cancel(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, "cancel", h.service.Cancel)
}
func (h *CampaignHandler) adminCancel(w http.ResponseWriter, r *http.Request) {
	c, err := h.service.AdminCancel(r.Context(), principal(r), chi.URLParam(r, "campaignID"), requestIDFromContext(r.Context()), remoteIP(r.RemoteAddr), r.UserAgent())
	if err != nil {
		h.handleCampaignError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaign": c})
}
func (h *CampaignHandler) adminList(w http.ResponseWriter, r *http.Request) {
	items, page, err := h.service.AdminList(r.Context(), principal(r), listQuery(r))
	if err != nil {
		h.handleCampaignError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaigns": items, "pagination": page})
}
func (h *CampaignHandler) adminGet(w http.ResponseWriter, r *http.Request) {
	c, err := h.service.AdminGet(r.Context(), principal(r), chi.URLParam(r, "campaignID"))
	if err != nil {
		h.handleCampaignError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaign": c})
}

type transitionFunc func(context.Context, auth.Principal, string, string, string, string) (campaign.Campaign, error)

func (h *CampaignHandler) transition(w http.ResponseWriter, r *http.Request, _ string, fn transitionFunc) {
	c, err := fn(r.Context(), principal(r), chi.URLParam(r, "campaignID"), requestIDFromContext(r.Context()), remoteIP(r.RemoteAddr), r.UserAgent())
	if err != nil {
		h.handleCampaignError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaign": c})
}
func principal(r *http.Request) auth.Principal {
	p, _ := auth.PrincipalFromContext(r.Context())
	return p
}
func listQuery(r *http.Request) campaign.ListQuery {
	q := r.URL.Query()
	return campaign.ListQuery{Page: parseInt(q.Get("page")), Limit: parseInt(q.Get("limit")), Search: strings.TrimSpace(q.Get("search")), Status: q.Get("status"), Platform: q.Get("platform"), BrandID: q.Get("brand_id"), StartDate: q.Get("start_date"), EndDate: q.Get("end_date"), Sort: q.Get("sort"), Direction: q.Get("direction")}
}
func parseInt(value string) int {
	var n int
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
func (h *CampaignHandler) handleCampaignError(w http.ResponseWriter, r *http.Request, err error) {
	if domain := campaign.AsError(err); domain != nil {
		writeCampaignError(w, r, domain)
		return
	}
	if domain := auth.AsError(err); domain != nil {
		writeAuthError(w, r, domain)
		return
	}
	h.logger.ErrorContext(r.Context(), "campaign operation failed", "request_id", requestIDFromContext(r.Context()), "error", err)
	writeAPIError(w, r, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred.")
}
func writeCampaignError(w http.ResponseWriter, r *http.Request, err *campaign.Error) {
	writeAPIError(w, r, err.HTTPStatus, err.Code, err.Message)
}

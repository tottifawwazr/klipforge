package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/klipforge/klipforge/services/api/internal/youtube"
)

type YouTubeHandler struct{ service *youtube.Service }

func NewYouTubeHandler(service *youtube.Service) *YouTubeHandler {
	return &YouTubeHandler{service: service}
}
func (h *YouTubeHandler) index(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Index(r.Context(), principal(r), r.URL.Query().Get("campaign_id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, data)
}
func (h *YouTubeHandler) metrics(w http.ResponseWriter, r *http.Request) { h.load(w, r, false) }
func (h *YouTubeHandler) refresh(w http.ResponseWriter, r *http.Request) { h.load(w, r, true) }
func (h *YouTubeHandler) load(w http.ResponseWriter, r *http.Request, force bool) {
	data, err := h.service.Get(r.Context(), principal(r), chi.URLParam(r, "submissionID"), force)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"metrics": data})
}
func (h *YouTubeHandler) fail(w http.ResponseWriter, r *http.Request, err error) {
	status, code := 503, "YOUTUBE_UNAVAILABLE"
	switch {
	case errors.Is(err, youtube.ErrNotConfigured):
		code = "YOUTUBE_NOT_CONFIGURED"
	case errors.Is(err, youtube.ErrForbidden):
		status, code = 403, "FORBIDDEN"
	case errors.Is(err, youtube.ErrNotFound):
		status, code = 404, "RESOURCE_NOT_FOUND"
	case errors.Is(err, youtube.ErrInvalid):
		status, code = 400, "YOUTUBE_INVALID_VIDEO"
	case errors.Is(err, youtube.ErrNotApproved):
		status, code = 409, "SUBMISSION_NOT_APPROVED"
	case errors.Is(err, youtube.ErrVideoUnavailable):
		status, code = 422, "YOUTUBE_VIDEO_UNAVAILABLE"
	case errors.Is(err, youtube.ErrBusy):
		status, code = 429, "YOUTUBE_REQUEST_BUSY"
		w.Header().Set("Retry-After", "60")
	case errors.Is(err, youtube.ErrQuota):
		status, code = 429, "YOUTUBE_QUOTA_REACHED"
	}
	message := youtube.ErrUnavailable.Error()
	if status != 503 || errors.Is(err, youtube.ErrNotConfigured) {
		message = err.Error()
	}
	writeAPIError(w, r, status, code, message)
}

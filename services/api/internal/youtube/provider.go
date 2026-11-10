// Package youtube retrieves public counters separately from simulated campaign metrics.
package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotConfigured    = errors.New("YouTube metrics are not connected. Ask an admin to configure the server API key")
	ErrUnavailable      = errors.New("YouTube metrics are temporarily unavailable. Try again later")
	ErrQuota            = errors.New("YouTube request quota has been reached. Try again later")
	ErrVideoUnavailable = errors.New("YouTube did not return public statistics for this video")
	ErrInvalid          = errors.New("Choose an approved YouTube submission with a valid direct video URL")
	ErrNotFound         = errors.New("The submission could not be found")
	ErrForbidden        = errors.New("You do not have access to this operation")
	ErrNotApproved      = errors.New("Only approved YouTube submissions can load public metrics")
	ErrBusy             = errors.New("This video was requested recently. Wait a minute before trying again")
)

const cacheTTL = 15 * time.Minute

var videoPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

func VideoID(value string) (string, error) {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return "", ErrInvalid
	}
	host := strings.ToLower(u.Hostname())
	var id string
	switch host {
	case "youtu.be":
		id = strings.TrimPrefix(u.EscapedPath(), "/")
	case "youtube.com", "www.youtube.com", "m.youtube.com":
		if u.Path == "/watch" {
			id = u.Query().Get("v")
		} else {
			parts := strings.Split(strings.Trim(u.Path, "/"), "/")
			if len(parts) == 2 && (parts[0] == "shorts" || parts[0] == "embed" || parts[0] == "live") {
				id = parts[1]
			}
		}
	default:
		return "", ErrInvalid
	}
	if !videoPattern.MatchString(id) {
		return "", ErrInvalid
	}
	return id, nil
}

type Snapshot struct {
	Source         string    `json:"source"`
	VideoID        string    `json:"video_id"`
	Views          int64     `json:"views"`
	Likes          *int64    `json:"likes"`
	Comments       *int64    `json:"comments"`
	FetchedAt      time.Time `json:"fetched_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	Scope          string    `json:"scope"`
	UsedForPayouts bool      `json:"used_for_payouts"`
}
type Provider struct {
	key      string
	endpoint string
	client   *http.Client
}

func NewProvider(key string) *Provider {
	return &Provider{key: strings.TrimSpace(key), endpoint: "https://www.googleapis.com/youtube/v3/videos", client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}
func (p *Provider) Configured() bool { return p != nil && p.key != "" }
func (p *Provider) Fetch(ctx context.Context, id string) (Snapshot, error) {
	if !p.Configured() {
		return Snapshot{}, ErrNotConfigured
	}
	if !videoPattern.MatchString(id) {
		return Snapshot{}, ErrInvalid
	}
	endpoint, err := url.Parse(p.endpoint)
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	query := endpoint.Query()
	query.Set("part", "statistics")
	query.Set("id", id)
	query.Set("fields", "items(id,statistics(viewCount,likeCount,commentCount))")
	query.Set("key", p.key)
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	request.Header.Set("Accept", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1_048_577))
	if err != nil || len(body) > 1_048_576 {
		return Snapshot{}, ErrUnavailable
	}
	if response.StatusCode != http.StatusOK {
		var failure struct {
			Error struct {
				Errors []struct {
					Reason string `json:"reason"`
				} `json:"errors"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &failure)
		for _, item := range failure.Error.Errors {
			if item.Reason == "quotaExceeded" || item.Reason == "dailyLimitExceeded" || item.Reason == "rateLimitExceeded" {
				return Snapshot{}, ErrQuota
			}
		}
		if response.StatusCode == 429 {
			return Snapshot{}, ErrQuota
		}
		if response.StatusCode == 404 {
			return Snapshot{}, ErrVideoUnavailable
		}
		return Snapshot{}, ErrUnavailable
	}
	var data struct {
		Items []struct {
			ID         string `json:"id"`
			Statistics struct {
				Views    *string `json:"viewCount"`
				Likes    *string `json:"likeCount"`
				Comments *string `json:"commentCount"`
			} `json:"statistics"`
		} `json:"items"`
	}
	if json.Unmarshal(body, &data) != nil {
		return Snapshot{}, ErrUnavailable
	}
	if len(data.Items) == 0 {
		return Snapshot{}, ErrVideoUnavailable
	}
	if len(data.Items) != 1 || data.Items[0].ID != id {
		return Snapshot{}, ErrUnavailable
	}
	item := data.Items[0]
	views, err := counter(item.Statistics.Views)
	if err != nil || views == nil {
		return Snapshot{}, ErrVideoUnavailable
	}
	likes, err := counter(item.Statistics.Likes)
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	comments, err := counter(item.Statistics.Comments)
	if err != nil {
		return Snapshot{}, ErrUnavailable
	}
	now := time.Now().UTC()
	return Snapshot{Source: "YOUTUBE_DATA_API", VideoID: id, Views: *views, Likes: likes, Comments: comments, FetchedAt: now, ExpiresAt: now.Add(cacheTTL), Scope: "PUBLIC_LIFETIME_COUNTERS", UsedForPayouts: false}, nil
}
func counter(raw *string) (*int64, error) {
	if raw == nil {
		return nil, nil
	}
	if *raw == "" {
		return nil, ErrUnavailable
	}
	for _, c := range *raw {
		if c < '0' || c > '9' {
			return nil, ErrUnavailable
		}
	}
	n, err := strconv.ParseInt(*raw, 10, 64)
	if err != nil {
		return nil, ErrUnavailable
	}
	return &n, nil
}

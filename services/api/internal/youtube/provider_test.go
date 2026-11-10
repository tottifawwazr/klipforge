package youtube

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVideoIDs(t *testing.T) {
	for _, value := range []string{"https://www.youtube.com/watch?v=abcdefghijk&feature=share", "https://youtu.be/abcdefghijk?t=5", "https://m.youtube.com/shorts/abcdefghijk", "https://youtube.com/embed/abcdefghijk", "https://youtube.com/live/abcdefghijk"} {
		id, err := VideoID(value)
		if err != nil || id != "abcdefghijk" {
			t.Fatalf("%s: %q %v", value, id, err)
		}
	}
	for _, value := range []string{"http://youtu.be/abcdefghijk", "https://youtube.com.attacker.example/watch?v=abcdefghijk", "https://youtube.com@evil.example/watch?v=abcdefghijk", "https://youtu.be/abcdefghijk/extra", "https://youtube.com/watch?v=short", "https://www.youtube.com:444/watch?v=abcdefghijk", "file:///watch?v=abcdefghijk"} {
		if _, err := VideoID(value); !errors.Is(err, ErrInvalid) {
			t.Fatalf("unsafe URL accepted: %s", value)
		}
	}
}
func TestPublicCountersAndMissingFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "fixture-secret" || r.URL.Query().Get("part") != "statistics" || r.URL.Query().Get("id") != "abcdefghijk" {
			t.Error("unexpected request parameters")
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"items":[{"id":"abcdefghijk","statistics":{"viewCount":"1000","likeCount":"0"}}]}`)
	}))
	defer server.Close()
	provider := NewProvider("fixture-secret")
	provider.endpoint = server.URL
	metrics, err := provider.Fetch(context.Background(), "abcdefghijk")
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Views != 1000 || metrics.Likes == nil || *metrics.Likes != 0 || metrics.Comments != nil || metrics.UsedForPayouts || metrics.Source != "YOUTUBE_DATA_API" || metrics.ExpiresAt.Sub(metrics.FetchedAt) != 15*time.Minute {
		t.Fatalf("wrong public counters: %+v", metrics)
	}
}
func TestProviderFailuresAreSanitized(t *testing.T) {
	cases := []struct {
		name, body string
		status     int
		expected   error
	}{
		{"missing", `{"items":[]}`, 200, ErrVideoUnavailable},
		{"wrong video", `{"items":[{"id":"wrongwrongx","statistics":{"viewCount":"12"}}]}`, 200, ErrUnavailable},
		{"missing views", `{"items":[{"id":"abcdefghijk","statistics":{}}]}`, 200, ErrVideoUnavailable},
		{"negative counter", `{"items":[{"id":"abcdefghijk","statistics":{"viewCount":"12","likeCount":"-1"}}]}`, 200, ErrUnavailable},
		{"overflow", `{"items":[{"id":"abcdefghijk","statistics":{"viewCount":"999999999999999999999999999"}}]}`, 200, ErrVideoUnavailable},
		{"quota", `{"error":{"message":"fixture-secret","errors":[{"reason":"quotaExceeded"}]}}`, 403, ErrQuota},
		{"bad key", `{"error":{"message":"fixture-secret","errors":[{"reason":"keyInvalid"}]}}`, 400, ErrUnavailable},
		{"upstream", `fixture-secret`, 500, ErrUnavailable},
		{"too many requests", `fixture-secret`, 429, ErrQuota},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); fmt.Fprint(w, tc.body) }))
			defer server.Close()
			provider := NewProvider("fixture-secret")
			provider.endpoint = server.URL
			_, err := provider.Fetch(context.Background(), "abcdefghijk")
			if !errors.Is(err, tc.expected) || strings.Contains(fmt.Sprint(err), "fixture-secret") {
				t.Fatalf("unsafe or wrong error: %v", err)
			}
		})
	}
}
func TestProviderDoesNotFollowRedirects(t *testing.T) {
	targetCalls := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetCalls++ }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer redirect.Close()
	provider := NewProvider("fixture-secret")
	provider.endpoint = redirect.URL
	_, err := provider.Fetch(context.Background(), "abcdefghijk")
	if !errors.Is(err, ErrUnavailable) || targetCalls != 0 {
		t.Fatalf("redirect followed: %v calls=%d", err, targetCalls)
	}
}
func TestProviderHandlesAbsentKeyAndCancellation(t *testing.T) {
	if _, err := NewProvider("").Fetch(context.Background(), "abcdefghijk"); !errors.Is(err, ErrNotConfigured) {
		t.Fatal(err)
	}
	provider := NewProvider("fixture-secret")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := provider.Fetch(ctx, "abcdefghijk")
	if !errors.Is(err, ErrUnavailable) || strings.Contains(fmt.Sprint(err), "fixture-secret") {
		t.Fatalf("cancellation leaked a request: %v", err)
	}
}

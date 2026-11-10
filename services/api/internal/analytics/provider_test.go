package analytics

import (
	"testing"
	"time"
)

func TestMockMetricsDeterministicAndMonotonic(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for _, platform := range []string{"TIKTOK", "INSTAGRAM", "YOUTUBE"} {
		first, err := MockMetrics("clip-1", platform, start, start.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		again, _ := MockMetrics("clip-1", platform, start, start.Add(time.Hour))
		if first != again {
			t.Fatal("same input changed")
		}
		later, _ := MockMetrics("clip-1", platform, start, start.Add(48*time.Hour))
		if later.Views <= first.Views || later.Likes < first.Likes || later.Comments < first.Comments || later.Shares < first.Shares || later.Saves < first.Saves {
			t.Fatal("metrics decreased")
		}
		if first.Likes+first.Comments+first.Shares+first.Saves > first.Views {
			t.Fatal("invalid engagement counts")
		}
	}
}
func TestMockMetricsRejectInvalidInput(t *testing.T) {
	now := time.Now()
	for _, v := range []struct {
		id, platform string
		start        time.Time
	}{{"", "TIKTOK", now}, {"id", "OTHER", now}, {"id", "TIKTOK", now.Add(time.Hour)}, {"id", "TIKTOK", time.Time{}}} {
		if _, err := MockMetrics(v.id, v.platform, v.start, now); err == nil {
			t.Fatalf("accepted invalid input: %+v", v)
		}
	}
}
func TestIDValidation(t *testing.T) {
	for _, id := range []string{"", "not-a-uuid", "00000000-0000-0000-0000-00000000000z"} {
		if validID(id) {
			t.Fatal("invalid ID accepted")
		}
	}
}

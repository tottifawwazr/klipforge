// Package analytics collects explicitly simulated social metrics for the local MVP.
package analytics

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"
)

type Metrics struct {
	Views    int64 `json:"views"`
	Likes    int64 `json:"likes"`
	Comments int64 `json:"comments"`
	Shares   int64 `json:"shares"`
	Saves    int64 `json:"saves"`
}

// MockMetrics is deterministic, monotonic, and never contacts a social network.
// Submitted time is anchored to UTC; a future submission is rejected.
func MockMetrics(id, platform string, submitted, captured time.Time) (Metrics, error) {
	if id == "" || submitted.IsZero() || captured.Before(submitted) {
		return Metrics{}, fmt.Errorf("invalid metric input")
	}
	switch platform {
	case "TIKTOK", "INSTAGRAM", "YOUTUBE":
	default:
		return Metrics{}, fmt.Errorf("unsupported platform")
	}
	sum := sha256.Sum256([]byte(platform + ":" + id))
	base := int64(binary.BigEndian.Uint32(sum[:4])%900 + 100)
	growth := int64(binary.BigEndian.Uint32(sum[4:8])%160 + 20)
	hours := int64(captured.Sub(submitted) / time.Hour)
	if hours > 24*365*10 {
		hours = 24 * 365 * 10
	}
	views := base + hours*growth
	return Metrics{Views: views, Likes: views / 17, Comments: views / 190, Shares: views / 83, Saves: views / 127}, nil
}

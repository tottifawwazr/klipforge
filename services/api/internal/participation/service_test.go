package participation

import "testing"

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		raw      string
		want     string
		wantErr  error
	}{
		{"tiktok canonicalizes tracking", "TIKTOK", "https://WWW.TikTok.com/@ari/video/42/?utm_source=test#fragment", "https://tiktok.com/@ari/video/42", nil},
		{"instagram reel", "INSTAGRAM", "https://www.instagram.com/reel/ABC123/", "https://instagram.com/reel/ABC123", nil},
		{"youtube short", "YOUTUBE", "https://youtu.be/abc123?feature=share", "https://youtu.be/abc123", nil},
		{"credentials rejected", "TIKTOK", "https://user:pass@tiktok.com/@ari/video/42", "", ErrInvalidURL},
		{"wrong host rejected", "TIKTOK", "https://example.com/@ari/video/42", "", ErrInvalidURL},
		{"port rejected", "TIKTOK", "https://tiktok.com:443/@ari/video/42", "", ErrInvalidURL},
		{"http rejected", "TIKTOK", "http://tiktok.com/@ari/video/42", "", ErrInvalidURL},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeURL(test.raw, test.platform)
			if err != test.wantErr {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("url = %q, want %q", got, test.want)
			}
		})
	}
}

func TestListQueryValidation(t *testing.T) {
	if !validParticipationQuery(normalizeQuery(ListQuery{Status: "ACCEPTED", Sort: "joined_at", Direction: "asc"})) {
		t.Fatal("valid participation query rejected")
	}
	if validParticipationQuery(normalizeQuery(ListQuery{Status: "UNKNOWN"})) {
		t.Fatal("invalid participation status accepted")
	}
	if !validSubmissionQuery(normalizeQuery(ListQuery{Platform: "TIKTOK", Status: "PENDING", Sort: "submitted_at"})) {
		t.Fatal("valid submission query rejected")
	}
	if validSubmissionQuery(normalizeQuery(ListQuery{Sort: "content_url"})) {
		t.Fatal("non-allow-listed submission sort accepted")
	}
}

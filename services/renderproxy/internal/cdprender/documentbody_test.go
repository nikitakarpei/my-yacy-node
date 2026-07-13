package cdprender

import "testing"

func TestIsHypertext(t *testing.T) {
	cases := []struct {
		contentType string
		want        bool
	}{
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"application/xhtml+xml", true},
		{"TEXT/HTML", true},
		{"application/pdf", false},
		{"application/json", false},
		{"image/png", false},
		{"text/plain", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isHypertext(c.contentType); got != c.want {
			t.Errorf("isHypertext(%q) = %v, want %v", c.contentType, got, c.want)
		}
	}
}

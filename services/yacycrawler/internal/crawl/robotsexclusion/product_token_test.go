package robotsexclusion_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/robotsexclusion"
)

func TestTheProductTokenIsTheLeadingTokenOfTheUserAgent(t *testing.T) {
	for userAgent, want := range map[string]robotsexclusion.ProductToken{
		"yacycrawler (+https://yacy.net)": "yacycrawler",
		"yacybot/1.0":                     "yacybot",
		"my_crawler-two":                  "my_crawler-two",
	} {
		token, err := robotsexclusion.ProductTokenOf(userAgent)
		if err != nil || token != want {
			t.Errorf("ProductTokenOf(%q) = %q, %v, want %q", userAgent, token, err, want)
		}
	}
}

func TestAUserAgentWithoutAProductTokenIsRefused(t *testing.T) {
	for _, userAgent := range []string{"", "(compatible)", "crawler2/1.0", "crawler.bot"} {
		if token, err := robotsexclusion.ProductTokenOf(userAgent); err == nil {
			t.Errorf("ProductTokenOf(%q) = %q, want a refusal", userAgent, token)
		}
	}
}

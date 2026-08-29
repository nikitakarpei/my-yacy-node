package pagemarkdownstore

import (
	"encoding/json"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const ScrapeOutcomeSubjectPrefix = "page.markdown.scraped"

const EveryScrapeOutcomeSubject = ScrapeOutcomeSubjectPrefix + ".*"

type ScrapeOutcome string

const (
	MarkdownStored ScrapeOutcome = "markdown-stored"
	PageGivenUp    ScrapeOutcome = "page-given-up"
)

type ScrapeOutcomeNotice struct {
	RequestedURL canonicalurl.CanonicalURL `json:"RequestedURL"`
	Outcome      ScrapeOutcome             `json:"Outcome"`
}

func ScrapeOutcomeSubjectOf(requestedURL canonicalurl.CanonicalURL) string {
	return ScrapeOutcomeSubjectPrefix + "." + fingerprintOf(requestedURL)
}

func MarshalScrapeOutcomeNotice(notice ScrapeOutcomeNotice) ([]byte, error) {
	data, err := json.Marshal(notice)
	if err != nil {
		return nil, fmt.Errorf("marshal scrape outcome notice: %w", err)
	}
	return data, nil
}

func UnmarshalScrapeOutcomeNotice(data []byte) (ScrapeOutcomeNotice, error) {
	var notice ScrapeOutcomeNotice
	if err := json.Unmarshal(data, &notice); err != nil {
		return ScrapeOutcomeNotice{}, fmt.Errorf("unmarshal scrape outcome notice: %w", err)
	}
	if notice.Outcome != MarkdownStored && notice.Outcome != PageGivenUp {
		return ScrapeOutcomeNotice{},
			fmt.Errorf("unmarshal scrape outcome notice: unknown outcome %q", notice.Outcome)
	}
	return notice, nil
}

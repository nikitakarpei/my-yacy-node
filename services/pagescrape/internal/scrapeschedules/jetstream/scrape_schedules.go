package jetstream

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type ScrapeSchedules struct {
	scrapeRequests jetstream.JetStream
	clock          Clock
}

func NewScrapeSchedules(
	scrapeRequests jetstream.JetStream,
	clock Clock,
) *ScrapeSchedules {
	return &ScrapeSchedules{scrapeRequests: scrapeRequests, clock: clock}
}

func (s *ScrapeSchedules) ScheduleScrape(
	ctx context.Context,
	request pagescrapecontract.ScrapeRequest,
	after time.Duration,
) error {
	data, err := pagescrapecontract.MarshalScrapeRequest(request)
	if err != nil {
		return err
	}
	if _, err := s.scrapeRequests.Publish(
		ctx,
		pagescrapecontract.ScrapeScheduleSubjectOf(request.PageURL),
		data,
		jetstream.WithScheduleAt(s.clock.Now().Add(after)),
		jetstream.WithScheduleTarget(pagescrapecontract.ScrapeRequestSubject),
	); err != nil {
		return fmt.Errorf("schedule the scrape of %q in %s: %w", request.PageURL, after, err)
	}
	return nil
}

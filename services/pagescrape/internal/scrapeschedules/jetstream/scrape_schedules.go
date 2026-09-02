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
	readingTime    func() time.Time
}

func NewScrapeSchedules(
	scrapeRequests jetstream.JetStream,
	readingTime func() time.Time,
) *ScrapeSchedules {
	if readingTime == nil {
		readingTime = time.Now
	}
	return &ScrapeSchedules{scrapeRequests: scrapeRequests, readingTime: readingTime}
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
		jetstream.WithScheduleAt(s.readingTime().Add(after)),
		jetstream.WithScheduleTarget(pagescrapecontract.ScrapeRequestSubject),
	); err != nil {
		return fmt.Errorf("schedule the scrape of %q in %s: %w", request.PageURL, after, err)
	}
	return nil
}

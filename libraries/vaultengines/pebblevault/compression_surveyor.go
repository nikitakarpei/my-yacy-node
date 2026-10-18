package pebblevault

import (
	"context"
	"log/slog"
	"sync/atomic"

	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault/internal/compressionratios"
)

const tableChangesPerSurvey = 16

type compressionSurveyor struct {
	tableChanges atomic.Uint64
	surveyDue    chan struct{}
	stopping     chan struct{}
	stopped      chan struct{}
	latestRatios atomic.Pointer[compressionratios.Ratios]
}

func newCompressionSurveyor() *compressionSurveyor {
	surveyor := &compressionSurveyor{
		surveyDue: make(chan struct{}, 1),
		stopping:  make(chan struct{}),
		stopped:   make(chan struct{}),
	}
	surveyor.latestRatios.Store(&compressionratios.Ratios{})

	return surveyor
}

func (s *compressionSurveyor) eventListener() pebble.EventListener {
	return pebble.EventListener{
		FlushEnd:      func(pebble.FlushInfo) { s.noticeTablesChanged() },
		CompactionEnd: func(pebble.CompactionInfo) { s.noticeTablesChanged() },
	}
}

func (s *compressionSurveyor) noticeTablesChanged() {
	if s.tableChanges.Add(1)%tableChangesPerSurvey != 0 {
		return
	}
	select {
	case s.surveyDue <- struct{}{}:
	default:
	}
}

func (s *compressionSurveyor) start(db *pebble.DB) {
	s.survey(db)
	go s.surveyOnTableChanges(db)
}

func (s *compressionSurveyor) survey(db *pebble.DB) {
	tablesByLevel, err := db.SSTables(pebble.WithProperties())
	if err != nil {
		slog.WarnContext(
			context.Background(),
			"storage keeps its previous compression ratios",
			slog.Any("error", err),
		)

		return
	}
	ratios := compressionratios.Of(tablesByLevel)
	s.latestRatios.Store(&ratios)
	slog.DebugContext(context.Background(), "storage surveyed compression ratios")
}

func (s *compressionSurveyor) surveyOnTableChanges(db *pebble.DB) {
	defer close(s.stopped)

	for {
		select {
		case <-s.stopping:
			return
		case <-s.surveyDue:
			s.survey(db)
		}
	}
}

func (s *compressionSurveyor) diskBytesOf(tally compressionratios.BucketTally) int64 {
	return s.latestRatios.Load().DiskBytesOf(tally)
}

func (s *compressionSurveyor) stop() {
	close(s.stopping)
	<-s.stopped
}

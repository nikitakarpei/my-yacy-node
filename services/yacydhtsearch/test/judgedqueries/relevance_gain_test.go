package judgedqueries_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/sitediscount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

const (
	leastLiftOverFoundOrder        = 0.45
	toleranceBelowAcceptedMeanGain = 0.01
)

func TestTheOrderingOfTheServiceHoldsItsGainOverTheJudgedQueries(t *testing.T) {
	t.Parallel()

	queries := judgedQueriesRecorded(t)
	serviceOrder := queries.orderedBy(defaultServiceOrdering())
	gain := gainPerOrdering{
		service:   serviceOrder.gainPerQuery(),
		relevance: queries.gainPerQueryOf(defaultRelevanceOrdering()),
		found:     queries.gainPerQueryOf(foundOrder{}),
	}
	acceptedGain := acceptedGainPerQuery(t)
	gainOverSeveralRelevantDocuments := gain.over(queries.ofSeveralRelevantDocuments())

	reportGainPerQuery(t, serviceOrder, gain)
	reportMeanGainPerOrdering(t, gainOverSeveralRelevantDocuments)
	reportRelevantDocumentsHeld(t, queries.ofOneRelevantDocument(), gain.service)
	failOnAShortLift(t, gainOverSeveralRelevantDocuments)
	failOnAMeanGainBelowTheBaseline(t, gainOverSeveralRelevantDocuments.service, acceptedGain)
	failOnAQueryWithoutGain(t, gain.service, acceptedGain)
}

type gainPerOrdering struct {
	service   gainPerQuery
	relevance gainPerQuery
	found     gainPerQuery
}

func (gain gainPerOrdering) over(queries judgedQueries) gainPerOrdering {
	return gainPerOrdering{
		service:   gain.service.over(queries),
		relevance: gain.relevance.over(queries),
		found:     gain.found.over(queries),
	}
}

func defaultServiceOrdering() sitediscount.Ordering {
	return serviceOrderingWeighedBy(documentrelevance.DefaultRelevanceWeights())
}

func serviceOrderingWeighedBy(
	relevanceWeights documentrelevance.RelevanceWeights,
) sitediscount.Ordering {
	return sitediscount.New(documentrelevance.RelevanceScorerWeighedBy(relevanceWeights))
}

func defaultRelevanceOrdering() relevance.Ordering {
	return relevance.New(
		documentrelevance.RelevanceScorerWeighedBy(documentrelevance.DefaultRelevanceWeights()),
	)
}

func reportGainPerQuery(t *testing.T, serviceOrder orderedQueries, gain gainPerOrdering) {
	t.Helper()

	amountOfSpamDocumentsAmongTheFirst := 0
	for _, orderedQuery := range serviceOrder {
		query := orderedQuery.judgedQuery.query
		gradedDocuments := orderedQuery.judgedQuery.gradedDocuments
		amountOfSpamDocuments := gradedDocuments.
			amountOfSpamDocumentsAmong(theFirstOf(orderedQuery.orderedDocuments))
		amountOfSpamDocumentsAmongTheFirst += amountOfSpamDocuments
		t.Logf(
			"%q: site discount %.4f, relevance %.4f, found order %.4f, %d ungraded documents "+
				"dropped, %d spam documents in the first %d",
			query,
			gain.service[query],
			gain.relevance[query],
			gain.found[query],
			gradedDocuments.amountOfUngradedDocumentsAmong(orderedQuery.orderedDocuments),
			amountOfSpamDocuments,
			judgedDocumentsCeiling,
		)
	}
	t.Logf(
		"the ordering of the service puts %d spam documents in the first %d over %d judged "+
			"queries",
		amountOfSpamDocumentsAmongTheFirst,
		judgedDocumentsCeiling,
		len(serviceOrder),
	)
	t.Logf(
		"the ordering of the service reaches no gain on %d of %d judged queries",
		gain.service.amountOfQueriesWithoutGain(),
		len(serviceOrder),
	)
}

type foundOrder struct{}

func (foundOrder) OrderedDocumentsOf(
	answers queryanswers.AnsweredQuery,
) []queryanswers.FoundDocument {
	return answers.FoundDocuments
}

func reportMeanGainPerOrdering(t *testing.T, gainOverSeveralRelevantDocuments gainPerOrdering) {
	t.Helper()

	t.Logf(
		"the mean over the %d judged queries of several relevant documents: site discount "+
			"%.4f, relevance %.4f, found order %.4f",
		len(gainOverSeveralRelevantDocuments.service),
		gainOverSeveralRelevantDocuments.service.meanGain(),
		gainOverSeveralRelevantDocuments.relevance.meanGain(),
		gainOverSeveralRelevantDocuments.found.meanGain(),
	)
}

func reportRelevantDocumentsHeld(
	t *testing.T,
	queriesOfOneRelevantDocument judgedQueries,
	gainOfServiceOrdering gainPerQuery,
) {
	t.Helper()

	amountOfDocumentsHeldAmongTheFirst := 0
	for _, judgedQuery := range queriesOfOneRelevantDocument {
		if gainOfServiceOrdering[judgedQuery.query] > 0 {
			amountOfDocumentsHeldAmongTheFirst++
		}
	}
	t.Logf(
		"the ordering of the service holds the one relevant document of %d of %d judged "+
			"queries in the first %d, whose gain measures its place and not an order",
		amountOfDocumentsHeldAmongTheFirst,
		len(queriesOfOneRelevantDocument),
		judgedDocumentsCeiling,
	)
}

func failOnAShortLift(t *testing.T, gain gainPerOrdering) {
	t.Helper()

	meanGainOfServiceOrdering := gain.service.meanGain()
	meanGainOfFoundOrder := gain.found.meanGain()
	if meanGainOfServiceOrdering-meanGainOfFoundOrder >= leastLiftOverFoundOrder {
		return
	}
	t.Errorf(
		"the ordering of the service lifts the mean gain from %.4f to %.4f over %d judged "+
			"queries, want a lift of at least %.2f",
		meanGainOfFoundOrder,
		meanGainOfServiceOrdering,
		len(gain.service),
		leastLiftOverFoundOrder,
	)
}

func failOnAMeanGainBelowTheBaseline(
	t *testing.T, gainOfServiceOrdering, acceptedGain gainPerQuery,
) {
	t.Helper()

	meanGainOfServiceOrdering := gainOfServiceOrdering.meanGainSharedWith(acceptedGain)
	baselineMeanGain := acceptedGain.meanGainSharedWith(gainOfServiceOrdering)
	if meanGainOfServiceOrdering >= baselineMeanGain-toleranceBelowAcceptedMeanGain {
		return
	}
	t.Errorf(
		"the ordering of the service reaches a mean gain of %.4f over the judged queries of "+
			"the baseline, want at least the accepted mean %.4f less the tolerance %.2f",
		meanGainOfServiceOrdering,
		baselineMeanGain,
		toleranceBelowAcceptedMeanGain,
	)
}

func failOnAQueryWithoutGain(
	t *testing.T, gainOfServiceOrdering, acceptedGain gainPerQuery,
) {
	t.Helper()

	for _, query := range slices.Sorted(maps.Keys(gainOfServiceOrdering)) {
		accepted, inTheBaseline := acceptedGain[query]
		if !inTheBaseline || accepted == 0 || gainOfServiceOrdering[query] > 0 {
			continue
		}
		t.Errorf(
			"the ordering of the service reaches no gain on %q, the baseline accepts %.4f",
			query,
			accepted,
		)
	}
}

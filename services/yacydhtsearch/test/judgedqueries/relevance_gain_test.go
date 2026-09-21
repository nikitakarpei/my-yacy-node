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
	serviceOrdering := defaultServiceOrdering()
	gainOfServiceOrdering := queries.gainPerQueryOf(serviceOrdering)
	acceptedGain := acceptedGainPerQuery(t)
	queriesOfSeveralRelevantDocuments := queries.ofSeveralRelevantDocuments()

	reportGainPerQuery(t, queries, gainOfServiceOrdering)
	reportMeanGainPerOrdering(t, queriesOfSeveralRelevantDocuments)
	reportRelevantDocumentsHeld(t, queries.ofOneRelevantDocument(), gainOfServiceOrdering)
	failOnAShortLift(t, queriesOfSeveralRelevantDocuments)
	failOnAMeanGainBelowTheBaseline(
		t, queriesOfSeveralRelevantDocuments.gainPerQueryOf(serviceOrdering), acceptedGain,
	)
	failOnAQueryWithoutGain(t, gainOfServiceOrdering, acceptedGain)
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

func reportGainPerQuery(
	t *testing.T, queries judgedQueries, gainOfServiceOrdering gainPerQuery,
) {
	t.Helper()

	relevanceOrdering := defaultRelevanceOrdering()
	serviceOrdering := defaultServiceOrdering()
	amountOfSpamDocumentsAmongTheFirst := 0
	for _, judgedQuery := range queries {
		orderedDocuments := serviceOrdering.OrderedDocumentsOf(judgedQuery.answers)
		amountOfSpamDocumentsAmongTheFirst += judgedQuery.gradedDocuments.
			amountOfSpamDocumentsAmong(theFirstOf(orderedDocuments))
		t.Logf(
			"%q: site discount %.4f, relevance %.4f, found order %.4f, %d ungraded documents "+
				"dropped, %d spam documents in the first %d",
			judgedQuery.query,
			gainOfServiceOrdering[judgedQuery.query],
			judgedQuery.gradedDocuments.normalizedGainOf(
				relevanceOrdering.OrderedDocumentsOf(judgedQuery.answers),
			),
			judgedQuery.gradedDocuments.normalizedGainOf(
				foundOrder{}.OrderedDocumentsOf(judgedQuery.answers),
			),
			judgedQuery.gradedDocuments.amountOfUngradedDocumentsAmong(orderedDocuments),
			judgedQuery.gradedDocuments.amountOfSpamDocumentsAmong(theFirstOf(orderedDocuments)),
			judgedDocumentsCeiling,
		)
	}
	t.Logf(
		"the ordering of the service puts %d spam documents in the first %d over %d judged "+
			"queries",
		amountOfSpamDocumentsAmongTheFirst,
		judgedDocumentsCeiling,
		len(queries),
	)
	t.Logf(
		"the ordering of the service reaches no gain on %d of %d judged queries",
		gainOfServiceOrdering.amountOfQueriesWithoutGain(),
		len(queries),
	)
}

type foundOrder struct{}

func (foundOrder) OrderedDocumentsOf(
	answers queryanswers.AnsweredQuery,
) []queryanswers.FoundDocument {
	return answers.FoundDocuments
}

func reportMeanGainPerOrdering(t *testing.T, queriesOfSeveralRelevantDocuments judgedQueries) {
	t.Helper()

	t.Logf(
		"the mean over the %d judged queries of several relevant documents: site discount "+
			"%.4f, relevance %.4f, found order %.4f",
		len(queriesOfSeveralRelevantDocuments),
		queriesOfSeveralRelevantDocuments.meanGainOf(defaultServiceOrdering()),
		queriesOfSeveralRelevantDocuments.meanGainOf(defaultRelevanceOrdering()),
		queriesOfSeveralRelevantDocuments.meanGainOf(foundOrder{}),
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

func failOnAShortLift(t *testing.T, queries judgedQueries) {
	t.Helper()

	meanGainOfServiceOrdering := queries.meanGainOf(defaultServiceOrdering())
	meanGainOfFoundOrder := queries.meanGainOf(foundOrder{})
	if meanGainOfServiceOrdering-meanGainOfFoundOrder >= leastLiftOverFoundOrder {
		return
	}
	t.Errorf(
		"the ordering of the service lifts the mean gain from %.4f to %.4f over %d judged "+
			"queries, want a lift of at least %.2f",
		meanGainOfFoundOrder,
		meanGainOfServiceOrdering,
		len(queries),
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

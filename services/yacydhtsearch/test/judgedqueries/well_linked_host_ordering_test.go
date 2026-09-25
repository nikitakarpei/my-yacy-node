package judgedqueries_test

import (
	"os"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/sitediscount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/welllinkedhostrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/welllinkedhosts/binaryfuse"
)

const wellLinkedHostsFile = "testdata/well-linked-hosts.txt"

func TestResultsOfHostsOffTheWellLinkedListRankLowerWithFewerSpamAndNoLessGain(t *testing.T) {
	t.Parallel()

	queries := judgedQueriesRecorded(t)
	serviceOrder := queries.orderedBy(defaultServiceOrdering())
	wellLinkedHostOrder := queries.orderedBy(wellLinkedHostOrdering(t))
	queriesOfSeveralRelevantDocuments := queries.ofSeveralRelevantDocuments()
	meanGainOfServiceOrdering := serviceOrder.gainPerQuery().
		over(queriesOfSeveralRelevantDocuments).meanGain()
	meanGainOfWellLinkedHostOrdering := wellLinkedHostOrder.gainPerQuery().
		over(queriesOfSeveralRelevantDocuments).meanGain()
	amountOfSpamDocumentsOfServiceOrdering := serviceOrder.amountOfSpamDocumentsAmongTheFirst()
	amountOfSpamDocumentsOfWellLinkedHostOrdering := wellLinkedHostOrder.
		amountOfSpamDocumentsAmongTheFirst()

	t.Logf(
		"over the %d judged queries of several relevant documents the well-linked hosts "+
			"move the mean gain from %.4f to %.4f, and the spam documents in the first %d "+
			"from %d to %d over %d judged queries",
		len(queriesOfSeveralRelevantDocuments),
		meanGainOfServiceOrdering,
		meanGainOfWellLinkedHostOrdering,
		judgedDocumentsCeiling,
		amountOfSpamDocumentsOfServiceOrdering,
		amountOfSpamDocumentsOfWellLinkedHostOrdering,
		len(queries),
	)
	if meanGainOfWellLinkedHostOrdering < meanGainOfServiceOrdering-toleranceBelowAcceptedMeanGain {
		t.Errorf(
			"the well-linked hosts lower the mean gain from %.4f to %.4f, want at most %.2f less",
			meanGainOfServiceOrdering,
			meanGainOfWellLinkedHostOrdering,
			toleranceBelowAcceptedMeanGain,
		)
	}
	if amountOfSpamDocumentsOfWellLinkedHostOrdering >= amountOfSpamDocumentsOfServiceOrdering {
		t.Errorf(
			"the well-linked hosts leave %d spam documents in the first %d, want fewer than %d",
			amountOfSpamDocumentsOfWellLinkedHostOrdering,
			judgedDocumentsCeiling,
			amountOfSpamDocumentsOfServiceOrdering,
		)
	}
}

func wellLinkedHostOrdering(t *testing.T) sitediscount.Ordering {
	t.Helper()

	hostList, err := os.Open(wellLinkedHostsFile)
	if err != nil {
		t.Fatalf("open the well-linked hosts: %v", err)
	}
	defer func() { _ = hostList.Close() }()
	wellLinkedHosts, err := binaryfuse.New(hostList)
	if err != nil {
		t.Fatalf("read the well-linked hosts: %v", err)
	}

	return sitediscount.New(welllinkedhostrelevance.New(
		documentrelevance.RelevanceScorerWeighedBy(documentrelevance.DefaultRelevanceWeights()),
		wellLinkedHosts,
		welllinkedhostrelevance.RelevanceObservers{},
	))
}

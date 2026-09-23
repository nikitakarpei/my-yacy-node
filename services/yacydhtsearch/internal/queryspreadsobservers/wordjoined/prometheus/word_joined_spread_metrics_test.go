package prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	prometheusclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	queryspreadsobserverswordjoinedprometheus "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreadsobservers/wordjoined/prometheus"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func publishedBy(t *testing.T, registry *prometheusclient.Registry) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil),
	)

	return recorder.Body.String()
}

func TestOneWordJoinedSpreadPublishesWhatTheJoinFound(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), wordjoined.PerformedWordJoinedSpread{
		DiscoveryRound: wordjoined.PerformedDiscoveryRound{
			AmountOfQueryWords:                      4,
			AmountOfQueryWordsHeldByNoPeer:          1,
			AmountOfQueryWordsWithCompleteAbstracts: 2,
			AmountOfPeersWithANonEmptyAbstract:      4,
			LeadingQueryWordChoice:                  wordjoined.MoreCommonQueryWordWithCompleteAbstracts,
			AmountOfDocumentsOfTheLeadingQueryWord:  12,
		},
		CrossCheckRound: wordjoined.PerformedCrossCheckRound{
			AmountOfDocumentsSentForCrossChecking:           9,
			AmountOfCrossCheckCandidatesNoPeerTook:          3,
			AmountOfJoinedDocumentsFoundOnlyByCrossChecking: 6,
			AmountOfJoinedDocuments:                         10,
		},
		URLMetadataRound: wordjoined.PerformedURLMetadataRound{
			AmountOfJoinedDocumentsWithMetadata:   2,
			AmountOfLookedUpDocuments:             4,
			AmountOfLookedUpDocumentsWithMetadata: 3,
		},
		TimeSpent: 250 * time.Millisecond,
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spreads_total{join="documents",leading_query_word_choice="more common word, complete abstracts"} 1`,
		"yacydhtsearch_word_joined_spread_query_words_with_complete_abstracts_ratio_sum 0.5",
		"yacydhtsearch_word_joined_spread_joined_documents_found_only_by_cross_checking_ratio_sum 0.6",
		"yacydhtsearch_word_joined_spread_cross_check_candidates_no_peer_took_ratio_sum 0.25",
		"yacydhtsearch_word_joined_spread_joined_documents_dropped_before_metadata_lookup_ratio_sum 0.5",
		"yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio_sum 0.25",
		"yacydhtsearch_word_joined_spread_unheld_query_words_ratio_sum 0.25",
		"yacydhtsearch_word_joined_spread_duration_seconds_sum 0.25",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestEveryKindOfWordJoinedSpreadIsPublishedBeforeTheFirstSpread(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spreads_total{join="documents",leading_query_word_choice="rarest word, partial abstracts"} 0`,
		`yacydhtsearch_word_joined_spreads_total{join="documents",leading_query_word_choice="more common word, complete abstracts"} 0`,
		`yacydhtsearch_word_joined_spreads_total{join="documents",leading_query_word_choice="rarest word, complete abstracts"} 0`,
		`yacydhtsearch_word_joined_spreads_total{join="no document",leading_query_word_choice="rarest word, partial abstracts"} 0`,
		`yacydhtsearch_word_joined_spreads_total{join="no document",leading_query_word_choice="more common word, complete abstracts"} 0`,
		`yacydhtsearch_word_joined_spreads_total{join="no document",leading_query_word_choice="rarest word, complete abstracts"} 0`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func spreadOfQueryWords(amountOfQueryWords int) wordjoined.PerformedWordJoinedSpread {
	return wordjoined.PerformedWordJoinedSpread{
		DiscoveryRound: wordjoined.PerformedDiscoveryRound{
			AmountOfQueryWords:     amountOfQueryWords,
			LeadingQueryWordChoice: wordjoined.RarestQueryWordWithoutCompleteAbstracts,
		},
	}
}

func TestASpreadThatCrossCheckedNoDocumentPublishesNoShareOfCrossChecking(
	t *testing.T,
) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), spreadJoiningDocuments(3))

	body := publishedBy(t, registry)
	for _, published := range []string{
		"yacydhtsearch_word_joined_spread_joined_documents_found_only_by_cross_checking_ratio_count 0",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf(
				"metrics carry %q for a spread that cross-checked no document:\n%s",
				published,
				body,
			)
		}
	}
}

func TestASpreadNoDocumentHeldAllQueryWordsForIsCountedApart(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), spreadOfQueryWords(3))

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spreads_total{join="no document",leading_query_word_choice="rarest word, partial abstracts"} 1`,
		"yacydhtsearch_word_joined_spread_joined_documents_dropped_before_metadata_lookup_ratio_count 0",
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestASpreadWhoseDocumentsAllCarriedMetadataPublishesNoShareDroppedBeforeLookup(
	t *testing.T,
) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	spread := spreadJoiningDocuments(4)
	spread.URLMetadataRound.AmountOfJoinedDocumentsWithMetadata = 4
	metrics.WordJoinedSpreadPerformed(t.Context(), spread)

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		"yacydhtsearch_word_joined_spread_joined_documents_dropped_before_metadata_lookup_ratio_count 0",
	) {
		t.Fatalf("metrics carry a share dropped before lookup for a spread missing none:\n%s", body)
	}
}

func spreadJoiningDocuments(amountOfJoinedDocuments int) wordjoined.PerformedWordJoinedSpread {
	spread := spreadOfQueryWords(2)
	spread.CrossCheckRound.AmountOfJoinedDocuments = amountOfJoinedDocuments

	return spread
}

func TestASpreadThatLookedUpNoMetadataPublishesNoShareWithoutMetadata(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), spreadJoiningDocuments(4))

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		"yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio_count 0",
	) {
		t.Fatalf("metrics carry a share for a spread that looked up no metadata:\n%s", body)
	}
}

func TestASpreadWithNoDocumentToCrossCheckPublishesNoShareOfCandidatesNoPeerTook(
	t *testing.T,
) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), spreadOfQueryWords(2))

	body := publishedBy(t, registry)
	if !strings.Contains(
		body,
		"yacydhtsearch_word_joined_spread_cross_check_candidates_no_peer_took_ratio_count 0",
	) {
		t.Fatalf(
			"metrics carry a share of candidates no peer took for a spread with no "+
				"document to cross-check:\n%s",
			body,
		)
	}
}

func spreadJudgingThePeers(
	peerStandings []peerjudgements.PeerStanding,
	judgedPeers []peerjudgements.JudgedPeer,
) wordjoined.PerformedWordJoinedSpread {
	spread := spreadOfQueryWords(2)
	spread.PeerStandings = peerStandings
	spread.CrossCheckRound.JudgedPeers = judgedPeers

	return spread
}

func TestTheStandingAndTheJudgementOfAPeerAreCountedUnderTheQuestion(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), spreadJudgingThePeers(
		[]peerjudgements.PeerStanding{
			{Standing: peerjudgements.NeverJudged},
			{Standing: peerjudgements.Ignoring},
			{Standing: peerjudgements.NeverJudged},
		},
		[]peerjudgements.JudgedPeer{
			{Judgement: peerjudgements.Honored},
			{Judgement: peerjudgements.NoEvidence},
		},
	))

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spread_peer_standings_total{question="abstract holds only the documents to match",standing="never judged"} 2`,
		`yacydhtsearch_word_joined_spread_peer_standings_total{question="abstract holds only the documents to match",standing="ignoring"} 1`,
		`yacydhtsearch_word_joined_spread_peer_judgements_total{judged="honored",question="abstract holds only the documents to match"} 1`,
		`yacydhtsearch_word_joined_spread_peer_judgements_total{judged="no evidence",question="abstract holds only the documents to match"} 1`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestEveryStandingAndEveryJudgementIsPublishedBeforeTheFirstSpread(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spread_peer_standings_total{question="abstract holds only the documents to match",standing="honoring"} 0`,
		`yacydhtsearch_word_joined_spread_peer_standings_total{question="abstract holds only the documents to match",standing="ignoring"} 0`,
		`yacydhtsearch_word_joined_spread_peer_standings_total{question="abstract holds only the documents to match",standing="never judged"} 0`,
		`yacydhtsearch_word_joined_spread_peer_standings_total{question="abstract holds only the documents to match",standing="version changed"} 0`,
		`yacydhtsearch_word_joined_spread_peer_standings_total{question="abstract holds only the documents to match",standing="interval passed"} 0`,
		`yacydhtsearch_word_joined_spread_peer_judgements_total{judged="honored",question="abstract holds only the documents to match"} 0`,
		`yacydhtsearch_word_joined_spread_peer_judgements_total{judged="ignored",question="abstract holds only the documents to match"} 0`,
		`yacydhtsearch_word_joined_spread_peer_judgements_total{judged="no evidence",question="abstract holds only the documents to match"} 0`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestEachRoundThatAskedPublishesTheTimeItTookAndTheTimeItWasGiven(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), wordjoined.PerformedWordJoinedSpread{
		DiscoveryRound: wordjoined.PerformedDiscoveryRound{
			LeadingQueryWordChoice: wordjoined.RarestQueryWordWithCompleteAbstracts,
			Time: yacymodel.Some(wordjoined.RoundTime{
				Budget:    yacymodel.Some(1500 * time.Millisecond),
				TimeSpent: 500 * time.Millisecond,
			}),
		},
		CrossCheckRound: wordjoined.PerformedCrossCheckRound{
			Time: yacymodel.Some(wordjoined.RoundTime{
				Budget:    yacymodel.Some(2 * time.Second),
				TimeSpent: 2 * time.Second,
			}),
		},
		URLMetadataRound: wordjoined.PerformedURLMetadataRound{
			Time: yacymodel.Some(wordjoined.RoundTime{
				Budget:    yacymodel.Some(1250 * time.Millisecond),
				TimeSpent: 250 * time.Millisecond,
			}),
		},
	})

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spread_round_duration_seconds_sum{round="discovery"} 0.5`,
		`yacydhtsearch_word_joined_spread_round_budget_seconds_sum{round="discovery"} 1.5`,
		`yacydhtsearch_word_joined_spread_round_duration_seconds_sum{round="cross-check"} 2`,
		`yacydhtsearch_word_joined_spread_round_budget_seconds_sum{round="cross-check"} 2`,
		`yacydhtsearch_word_joined_spread_round_duration_seconds_sum{round="URL metadata"} 0.25`,
		`yacydhtsearch_word_joined_spread_round_budget_seconds_sum{round="URL metadata"} 1.25`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestARoundThatAskedNoPeerPublishesNoTime(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	metrics.WordJoinedSpreadPerformed(t.Context(), spreadOfQueryWords(2))

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spread_round_duration_seconds_count{round="discovery"} 0`,
		`yacydhtsearch_word_joined_spread_round_budget_seconds_count{round="discovery"} 0`,
		`yacydhtsearch_word_joined_spread_round_duration_seconds_count{round="cross-check"} 0`,
		`yacydhtsearch_word_joined_spread_round_budget_seconds_count{round="cross-check"} 0`,
		`yacydhtsearch_word_joined_spread_round_duration_seconds_count{round="URL metadata"} 0`,
		`yacydhtsearch_word_joined_spread_round_budget_seconds_count{round="URL metadata"} 0`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q for rounds that asked no peer:\n%s", published, body)
		}
	}
}

func TestARoundOfAQueryWithoutADeadlinePublishesItsTimeButNoBudget(t *testing.T) {
	t.Parallel()

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 5*time.Second)

	spread := spreadOfQueryWords(2)
	spread.DiscoveryRound.Time = yacymodel.Some(wordjoined.RoundTime{
		Budget:    yacymodel.None[time.Duration](),
		TimeSpent: 500 * time.Millisecond,
	})
	metrics.WordJoinedSpreadPerformed(t.Context(), spread)

	body := publishedBy(t, registry)
	for _, published := range []string{
		`yacydhtsearch_word_joined_spread_round_duration_seconds_sum{round="discovery"} 0.5`,
		`yacydhtsearch_word_joined_spread_round_budget_seconds_count{round="discovery"} 0`,
	} {
		if !strings.Contains(body, published) {
			t.Fatalf("metrics do not carry %q:\n%s", published, body)
		}
	}
}

func TestTheTimeOfARoundIsPublishedInABucketAtMostAQuarterAboveIt(t *testing.T) {
	t.Parallel()

	const timeSpent = 4 * time.Second

	registry := prometheusclient.NewRegistry()
	metrics := queryspreadsobserverswordjoinedprometheus.New(registry, 15*time.Second)

	spread := spreadOfQueryWords(2)
	spread.DiscoveryRound.Time = yacymodel.Some(wordjoined.RoundTime{
		Budget:    yacymodel.Some(timeSpent),
		TimeSpent: timeSpent,
	})
	metrics.WordJoinedSpreadPerformed(t.Context(), spread)

	upperEdge := upperEdgeOfTheFirstBucketHolding(
		t, publishedBy(t, registry),
		`yacydhtsearch_word_joined_spread_round_duration_seconds_bucket{round="discovery",le="`,
	)
	if upperEdge > timeSpent.Seconds()*1.25 {
		t.Fatalf(
			"a round of %s was published under a bucket up to %gs, want at most a quarter above",
			timeSpent,
			upperEdge,
		)
	}
}

func upperEdgeOfTheFirstBucketHolding(t *testing.T, body string, bucketPrefix string) float64 {
	t.Helper()

	for line := range strings.Lines(body) {
		rest, isBucket := strings.CutPrefix(line, bucketPrefix)
		if !isBucket || !strings.HasSuffix(strings.TrimSpace(rest), "} 1") {
			continue
		}
		edge, _, _ := strings.Cut(rest, `"`)
		upperEdge, err := strconv.ParseFloat(edge, 64)
		if err != nil {
			t.Fatalf("bucket %q has no upper edge: %v", line, err)
		}

		return upperEdge
	}
	t.Fatalf("no bucket %q holds the round:\n%s", bucketPrefix, body)

	return 0
}

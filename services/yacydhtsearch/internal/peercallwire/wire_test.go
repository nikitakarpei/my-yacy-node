package peercallwire_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	responseLimit  = 1 << 20
	networkName    = "freeworld"
	ringPartitions = yacymodel.DHTRingPartitions(16)

	spreadBudgetOfTheTests   = 8 * time.Second
	peerCallBudgetOfTheTests = 4 * time.Second
	callsInFlightOfTheTests  = 48
)

type recordedOutcome struct {
	mutex                          sync.Mutex
	answeredURLMetadata            int
	amountOfDescribedDocuments     int
	answeredSearchDocuments        int
	amountOfDocumentsInTheAbstract int
	amountOfMatchedDocuments       int
	refused                        int
	unreachable                    int
	unreadable                     int
	cancelled                      int
	waitedForASlot                 int
	tookASlot                      int
	waitsBeforeTheSlotWasTaken     int
	askedFor                       peerasks.AskedFor
	spent                          time.Duration
}

func (r *recordedOutcome) PeerCallWaitsForASlot(
	_ context.Context,
	_ string,
	askedFor peerasks.AskedFor,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.waitedForASlot++
	r.askedFor = askedFor
}

func (r *recordedOutcome) PeerCallTookASlot(
	_ context.Context,
	_ string,
	askedFor peerasks.AskedFor,
	_ time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.tookASlot++
	r.waitsBeforeTheSlotWasTaken = r.waitedForASlot
	r.askedFor = askedFor
}

func (r *recordedOutcome) PeerAnsweredURLMetadata(
	_ context.Context,
	_ string,
	amountOfDescribedDocuments int,
	spent time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.answeredURLMetadata++
	r.amountOfDescribedDocuments = amountOfDescribedDocuments
	r.spent = spent
}

func (r *recordedOutcome) PeerSearchedDocuments(
	_ context.Context,
	_ string,
	amountOfDocumentsInTheAbstract int,
	amountOfMatchedDocuments int,
	spent time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.answeredSearchDocuments++
	r.amountOfDocumentsInTheAbstract = amountOfDocumentsInTheAbstract
	r.amountOfMatchedDocuments = amountOfMatchedDocuments
	r.spent = spent
}

func (r *recordedOutcome) PeerRefused(
	_ context.Context,
	_ string,
	askedFor peerasks.AskedFor,
	_ int,
	spent time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.refused++
	r.askedFor = askedFor
	r.spent = spent
}

func (r *recordedOutcome) PeerUnreachable(
	_ context.Context,
	_ string,
	askedFor peerasks.AskedFor,
	_ error,
	spent time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.unreachable++
	r.askedFor = askedFor
	r.spent = spent
}

func (r *recordedOutcome) PeerAnswerUnreadable(
	_ context.Context,
	_ string,
	askedFor peerasks.AskedFor,
	_ error,
	spent time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.unreadable++
	r.askedFor = askedFor
	r.spent = spent
}

func (r *recordedOutcome) PeerCallCancelled(
	_ context.Context,
	_ string,
	askedFor peerasks.AskedFor,
	spent time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.cancelled++
	r.askedFor = askedFor
	r.spent = spent
}

func wireTo(observer peercallwire.PeerCallObserver) peercallwire.Wire {
	return wireHolding(callsInFlightOfTheTests, observer)
}

func wireHolding(
	callsInFlight int,
	observer peercallwire.PeerCallObserver,
) peercallwire.Wire {
	return wireCalling(callsInFlight, peerCallBudgetOfTheTests, observer)
}

func wireSpendingAtMost(
	peerCallBudget time.Duration,
	observer peercallwire.PeerCallObserver,
) peercallwire.Wire {
	return wireCalling(callsInFlightOfTheTests, peerCallBudget, observer)
}

func wireCalling(
	callsInFlight int,
	peerCallBudget time.Duration,
	observer peercallwire.PeerCallObserver,
) peercallwire.Wire {
	return peercallwire.New(
		http.DefaultClient,
		peercallwire.SearchedNetwork{Name: networkName, RingPartitions: ringPartitions},
		peercallwire.PeerCallLimits{
			MaxResponseBytes:  responseLimit,
			PeerCallsInFlight: callsInFlight,
			PeerCallBudget:    peerCallBudget,
		},
		observer,
	)
}

func callWithin(t *testing.T, budget time.Duration) context.Context {
	t.Helper()

	ctx, endCall := context.WithTimeout(t.Context(), budget)
	t.Cleanup(endCall)

	return ctx
}

func matchedDocumentsOf(
	t *testing.T,
	observer peercallwire.PeerCallObserver,
	ask peerasks.SearchDocumentsAsk,
) ([]peerasks.MatchedDocument, bool) {
	t.Helper()

	answeredAsks := wireTo(observer).AskForSearchDocuments(
		callWithin(t, spreadBudgetOfTheTests), []peerasks.SearchDocumentsAsk{ask},
	)
	if len(answeredAsks) == 0 {
		return nil, false
	}

	return answeredAsks[0].MatchedDocuments, true
}

func searchDocumentsAnswerOf(
	t *testing.T,
	observer peercallwire.PeerCallObserver,
	ask peerasks.SearchDocumentsAsk,
) (peerasks.AnsweredSearchDocumentsAsk, bool) {
	t.Helper()

	answeredAsks := wireTo(observer).AskForSearchDocuments(
		callWithin(t, spreadBudgetOfTheTests), []peerasks.SearchDocumentsAsk{ask},
	)
	if len(answeredAsks) == 0 {
		return peerasks.AnsweredSearchDocumentsAsk{}, false
	}

	return answeredAsks[0], true
}

type peerRequests struct {
	received []yacyproto.SearchRequest
}

func peerAnswering(t *testing.T, body string, status int) (string, *peerRequests) {
	t.Helper()

	requests := &peerRequests{}
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, reader *http.Request) {
			if err := reader.ParseForm(); err == nil {
				request, err := yacyproto.ParseSearchRequest(reader.Context(), reader.PostForm)
				if err == nil {
					requests.received = append(requests.received, request)
				}
			}
			writer.WriteHeader(status)
			_, _ = writer.Write([]byte(body))
		},
	))
	t.Cleanup(server.Close)

	return server.URL, requests
}

func peerAt(address string) peerdirectory.AskablePeer {
	return peerdirectory.AskablePeer{Hash: yacymodel.WordHash("peer"), Address: address}
}

func searchAnswerHolding(t *testing.T, addresses ...string) string {
	t.Helper()

	resources := make([]yacyproto.SearchResource, 0, len(addresses))
	for _, address := range addresses {
		hash, err := yacymodel.URLHashOf(address)
		if err != nil {
			t.Fatalf("URLHashOf(%q): %v", address, err)
		}
		resources = append(resources, yacyproto.SearchResource{
			Metadata: yacymodel.URLMetadata{Hash: hash, Address: address, Title: "Weather"},
		})
	}

	return searchAnswerFrom(yacyproto.SearchResponse{
		Count:     len(resources),
		Resources: resources,
	})
}

func searchAnswerFrom(response yacyproto.SearchResponse) string {
	message := response.Encode()
	yacyproto.InjectResponseHeader(message, response.Version, response.Uptime)

	return message.Encode()
}

func mustParseURLHash(t *testing.T, raw string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.ParseURLHash(raw)
	if err != nil {
		t.Fatalf("ParseURLHash(%q): %v", raw, err)
	}

	return hash
}

func TestAPeerAnswerBecomesMatchedDocuments(t *testing.T) {
	t.Parallel()

	observer := &recordedOutcome{}
	address, _ := peerAnswering(
		t, searchAnswerHolding(t, "https://example.org/weather"), http.StatusOK,
	)

	matchedDocuments, replied := matchedDocumentsOf(t, observer, peerasks.SearchDocumentsAsk{
		Peer: peerAt(address),
	})

	if !replied || len(matchedDocuments) != 1 ||
		matchedDocuments[0].Metadata.Address != "https://example.org/weather" {
		t.Fatalf(
			"AskForSearchDocuments = %+v, %v, want the address the peer reported",
			matchedDocuments,
			replied,
		)
	}
	if observer.answeredSearchDocuments != 1 || observer.amountOfMatchedDocuments != 1 {
		t.Fatalf(
			"PeerSearchedDocuments reported %d times with %d matched documents, want once with one",
			observer.answeredSearchDocuments,
			observer.amountOfMatchedDocuments,
		)
	}
	if observer.spent <= 0 {
		t.Fatalf(
			"PeerSearchedDocuments reported %v spent, want the time the call took",
			observer.spent,
		)
	}
}

func TestASearchDocumentsAskCarriesTheQueryAndTheNetworkTheServiceSearches(t *testing.T) {
	t.Parallel()

	address, requests := peerAnswering(t, searchAnswerHolding(t), http.StatusOK)
	query := searchquery.QueryFrom("berlin -rain", "de")

	matchedDocumentsOf(t, &recordedOutcome{}, peerasks.SearchDocumentsAsk{
		Peer:          peerAt(address),
		Word:          query.WordHashes()[0],
		ExcludedWords: query.ExclusionHashes(),
		Language:      query.Language,
		ItemsCeiling:  10,
	})

	if len(requests.received) != 1 {
		t.Fatalf("the peer received %d requests, want one", len(requests.received))
	}
	request := requests.received[0]
	if request.NetworkName != networkName || request.Partitions != int(ringPartitions) {
		t.Fatalf("request = %+v, want the network facts the service searches", request)
	}
	grantedAnswerTime := time.Duration(request.Time) * time.Millisecond
	if grantedAnswerTime <= 0 || grantedAnswerTime >= peerCallBudgetOfTheTests {
		t.Fatalf(
			"the peer was granted %v of the %v the call had, want less than the service waits",
			grantedAnswerTime,
			peerCallBudgetOfTheTests,
		)
	}
	if len(request.Query) != 1 || request.Query[0] != yacymodel.WordHash("berlin") ||
		len(request.Exclude) != 1 || request.Language != "de" || request.Count != 10 {
		t.Fatalf("request = %+v, want the query the ask named", request)
	}
}

func TestAPeerIsGrantedTheCallBudgetLessTheMarginTheAnswerNeeds(t *testing.T) {
	t.Parallel()

	const (
		peerCallBudget   = 3 * time.Second
		answerMarginRoom = time.Second
	)
	address, requests := peerAnswering(t, searchAnswerHolding(t), http.StatusOK)

	wireSpendingAtMost(peerCallBudget, &recordedOutcome{}).AskForSearchDocuments(
		callWithin(t, spreadBudgetOfTheTests),
		[]peerasks.SearchDocumentsAsk{{Peer: peerAt(address)}},
	)

	if len(requests.received) != 1 {
		t.Fatalf("the peer received %d requests, want one", len(requests.received))
	}
	grantedAnswerTime := time.Duration(requests.received[0].Time) * time.Millisecond
	if grantedAnswerTime <= 0 || grantedAnswerTime > peerCallBudget-answerMarginRoom {
		t.Fatalf(
			"the peer was granted %v of the %v budget, want at most %v",
			grantedAnswerTime,
			peerCallBudget,
			peerCallBudget-answerMarginRoom,
		)
	}
}

func TestASearchDocumentsAskAsksForTheAbstractAndTheItemsOfOneWordOnly(t *testing.T) {
	t.Parallel()

	word := yacymodel.WordHash("berlin")
	document := mustParseURLHash(t, "bbbbbbAAAAAA")
	body := yacyproto.SearchResponse{
		IndexAbstract: map[yacymodel.Hash][]yacymodel.URLHash{word: {document}},
	}.Encode().Encode()
	address, requests := peerAnswering(t, body, http.StatusOK)
	observer := &recordedOutcome{}

	answeredAsk, replied := searchDocumentsAnswerOf(
		t,
		observer,
		peerasks.SearchDocumentsAsk{Peer: peerAt(address), Word: word, ItemsCeiling: 7},
	)

	if !replied || len(answeredAsk.Abstract) != 1 ||
		answeredAsk.Abstract[0] != document {
		t.Fatalf(
			"AskForSearchDocuments = %+v, %v, want the document in the abstract of the peer",
			answeredAsk.Abstract,
			replied,
		)
	}
	request := requests.received[0]
	if len(request.Abstracts.Hashes()) != 1 || request.Abstracts.Hashes()[0] != word ||
		len(request.Query) != 1 || request.Query[0] != word || request.Count != 7 {
		t.Fatalf("request = %+v, want the abstract and the items of the one word", request)
	}
	if observer.answeredSearchDocuments != 1 || observer.amountOfDocumentsInTheAbstract != 1 {
		t.Fatalf(
			"PeerSearchedDocuments reported %d times with %d documents in the abstract, want once with one",
			observer.answeredSearchDocuments,
			observer.amountOfDocumentsInTheAbstract,
		)
	}
}

func TestASearchDocumentsAnswerReadsTheItemsAndTheDocumentsThePeerHolds(t *testing.T) {
	t.Parallel()

	word := yacymodel.WordHash("berlin")
	withoutAPosting := mustParseURLHash(t, "bbbbbbAAAAAA")
	body := yacyproto.SearchResponse{
		Count: 2,
		Resources: []yacyproto.SearchResource{
			searchResourceWithAPosting(t, "https://example.org/berlin", word),
			{Metadata: yacymodel.URLMetadata{
				Hash: withoutAPosting, Address: "https://example.org/other",
			}},
		},
		IndexCount: map[yacymodel.Hash]int{word: 4096},
	}.Encode().Encode()
	address, _ := peerAnswering(t, body, http.StatusOK)

	answeredAsk, replied := searchDocumentsAnswerOf(
		t,
		&recordedOutcome{},
		peerasks.SearchDocumentsAsk{Peer: peerAt(address), Word: word},
	)

	if !replied || len(answeredAsk.MatchedDocuments) != 2 ||
		answeredAsk.MatchedDocuments[0].Metadata.Address != "https://example.org/berlin" {
		t.Fatalf("AskForSearchDocuments read %+v, want the documents the peer matched",
			answeredAsk.MatchedDocuments)
	}
	sentPosting, sent := answeredAsk.MatchedDocuments[0].Posting.Get()
	if !sent || sentPosting.Hits != 3 || answeredAsk.MatchedDocuments[1].Posting.Present() {
		t.Fatalf("the documents carry the postings %+v, want the one the peer sent",
			answeredAsk.MatchedDocuments)
	}
	documentsHeld, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get()
	if !counted || documentsHeld != 4096 {
		t.Fatalf(
			"the answer carries %d documents held for the word, want 4096",
			documentsHeld,
		)
	}
}

func searchResourceWithAPosting(
	t *testing.T,
	address string,
	word yacymodel.Hash,
) yacyproto.SearchResource {
	t.Helper()

	hash, err := yacymodel.URLHashOf(address)
	if err != nil {
		t.Fatalf("URLHashOf(%q): %v", address, err)
	}

	return yacyproto.SearchResource{
		Metadata: yacymodel.URLMetadata{Hash: hash, Address: address, Title: "Berlin"},
		Posting: yacymodel.Some(yacymodel.RWIPosting{
			WordHash: word,
			URLHash:  hash,
			Language: yacymodel.LanguageOfUndeclaredDocument,
			Hits:     3,
		}),
	}
}

func TestAPeerThatCountsNoDocumentForTheWordAnswersWithoutACount(t *testing.T) {
	t.Parallel()

	word := yacymodel.WordHash("berlin")
	body := yacyproto.SearchResponse{
		IndexAbstract: map[yacymodel.Hash][]yacymodel.URLHash{
			word: {mustParseURLHash(t, "bbbbbbAAAAAA")},
		},
	}.Encode().Encode()
	address, _ := peerAnswering(t, body, http.StatusOK)

	answeredAsk, replied := searchDocumentsAnswerOf(
		t,
		&recordedOutcome{},
		peerasks.SearchDocumentsAsk{Peer: peerAt(address), Word: word},
	)

	if _, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get(); !replied || counted {
		t.Fatalf("the answer carries %+v documents held for the word, want none",
			answeredAsk.AmountOfDocumentsHeldForTheWord)
	}
}

func TestASearchDocumentsAskReadsNoDocumentOfAnotherWord(t *testing.T) {
	t.Parallel()

	body := yacyproto.SearchResponse{
		IndexAbstract: map[yacymodel.Hash][]yacymodel.URLHash{
			yacymodel.WordHash("weather"): {mustParseURLHash(t, "bbbbbbAAAAAA")},
		},
	}.Encode().Encode()
	address, _ := peerAnswering(t, body, http.StatusOK)

	answeredAsk, replied := searchDocumentsAnswerOf(
		t,
		&recordedOutcome{},
		peerasks.SearchDocumentsAsk{
			Peer: peerAt(address),
			Word: yacymodel.WordHash("berlin"),
		},
	)

	if !replied || len(answeredAsk.Abstract) != 0 {
		t.Fatalf(
			"AskForSearchDocuments = %+v, want no document of the word it did not ask for",
			answeredAsk.Abstract,
		)
	}
}

func TestAPeerThatRefusesTheSearchYieldsNoMatchedDocuments(t *testing.T) {
	t.Parallel()

	observer := &recordedOutcome{}
	address, _ := peerAnswering(t, "", http.StatusServiceUnavailable)

	matchedDocuments, replied := matchedDocumentsOf(
		t,
		observer,
		peerasks.SearchDocumentsAsk{Peer: peerAt(address)},
	)

	if replied || len(matchedDocuments) != 0 || observer.refused != 1 {
		t.Fatalf(
			"AskForSearchDocuments = %+v with %d refusals, want none and one",
			matchedDocuments,
			observer.refused,
		)
	}
}

func TestAFailedCallReportsWhatItAskedThePeerFor(t *testing.T) {
	t.Parallel()

	address, _ := peerAnswering(t, "", http.StatusServiceUnavailable)

	refusedSearchDocumentsAsk := &recordedOutcome{}
	searchDocumentsAnswerOf(
		t,
		refusedSearchDocumentsAsk,
		peerasks.SearchDocumentsAsk{
			Peer: peerAt(address),
			Word: yacymodel.WordHash("berlin"),
		},
	)

	if refusedSearchDocumentsAsk.askedFor != peerasks.SearchDocuments {
		t.Fatalf(
			"the refusal named %q, want the search documents",
			refusedSearchDocumentsAsk.askedFor,
		)
	}
}

func TestAPeerThatHoldsNothingStillReplies(t *testing.T) {
	t.Parallel()

	address, _ := peerAnswering(t, searchAnswerHolding(t), http.StatusOK)

	matchedDocuments, replied := matchedDocumentsOf(
		t, &recordedOutcome{}, peerasks.SearchDocumentsAsk{Peer: peerAt(address)},
	)

	if !replied || len(matchedDocuments) != 0 {
		t.Fatalf(
			"AskForSearchDocuments = %+v, %v, want a reply that carries nothing",
			matchedDocuments,
			replied,
		)
	}
}

func TestAPeerThatCannotBeReachedYieldsNoMatchedDocuments(t *testing.T) {
	t.Parallel()

	observer := &recordedOutcome{}

	matchedDocuments, replied := matchedDocumentsOf(
		t, observer, peerasks.SearchDocumentsAsk{Peer: peerAt("http://127.0.0.1:1")},
	)

	if replied || len(matchedDocuments) != 0 || observer.unreachable != 1 {
		t.Fatalf(
			"AskForSearchDocuments = %+v with %d unreachable, want none and one",
			matchedDocuments,
			observer.unreachable,
		)
	}
}

type peerCalls struct {
	paths []string
	forms []url.Values
}

func peerAnsweringURLMetadata(t *testing.T, body string) (string, *peerCalls) {
	t.Helper()

	calls := &peerCalls{}
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, reader *http.Request) {
			calls.paths = append(calls.paths, reader.URL.Path)
			if err := reader.ParseForm(); err == nil {
				calls.forms = append(calls.forms, reader.PostForm)
			}
			_, _ = writer.Write([]byte(body))
		},
	))
	t.Cleanup(server.Close)

	return server.URL, calls
}

func TestAURLMetadataAskFetchesTheDocumentsItNames(t *testing.T) {
	t.Parallel()

	document := mustParseURLHash(t, "Q_ylfl--9bK5")
	address, _ := peerAnsweringURLMetadata(
		t,
		`<rss><yacy><response>ok</response></yacy><channel><item>`+
			`<title>Weather</title><link>https://example.org/weather</link>`+
			`<guid isPermaLink="false">Q_ylfl--9bK5</guid>`+
			`</item></channel></rss>`,
	)

	answeredAsks := wireTo(&recordedOutcome{}).AskForURLMetadata(
		t.Context(),
		[]peerasks.URLMetadataAsk{{
			Peer:      peerAt(address),
			Documents: []yacymodel.URLHash{document},
		}},
	)

	if len(answeredAsks) != 1 || len(answeredAsks[0].MetadataOfEachDocument) != 1 {
		t.Fatalf("AskForURLMetadata = %+v, want the one document the ask named", answeredAsks)
	}
	metadata := answeredAsks[0].MetadataOfEachDocument[0]
	if metadata.Hash != document || metadata.Address != "https://example.org/weather" {
		t.Fatalf("metadata = %+v, want the document the peer named", metadata)
	}
}

func TestAnAnsweredURLMetadataAskIsReportedAsTheMetadataItIs(t *testing.T) {
	t.Parallel()

	document := mustParseURLHash(t, "Q_ylfl--9bK5")
	address, _ := peerAnsweringURLMetadata(
		t,
		`<rss><yacy><response>ok</response></yacy><channel><item>`+
			`<title>Weather</title><link>https://example.org/weather</link>`+
			`<guid isPermaLink="false">Q_ylfl--9bK5</guid>`+
			`</item></channel></rss>`,
	)

	observer := &recordedOutcome{}
	wireTo(observer).AskForURLMetadata(
		t.Context(),
		[]peerasks.URLMetadataAsk{{
			Peer:      peerAt(address),
			Documents: []yacymodel.URLHash{document},
		}},
	)

	if observer.answeredURLMetadata != 1 || observer.amountOfDescribedDocuments != 1 {
		t.Fatalf(
			"PeerAnsweredURLMetadata reported %d times with %d documents, want once with one",
			observer.answeredURLMetadata,
			observer.amountOfDescribedDocuments,
		)
	}
	if observer.answeredSearchDocuments != 0 {
		t.Fatal("a metadata answer was reported as documents a peer matched")
	}
}

func TestAURLMetadataAskGoesToTheURLsEndpointWithTheNamesAndTheNetwork(t *testing.T) {
	t.Parallel()

	document := mustParseURLHash(t, "Q_ylfl--9bK5")
	address, calls := peerAnsweringURLMetadata(
		t,
		`<rss><yacy><response>ok</response></yacy><channel></channel></rss>`,
	)

	wireTo(&recordedOutcome{}).AskForURLMetadata(
		t.Context(),
		[]peerasks.URLMetadataAsk{{
			Peer:      peerAt(address),
			Documents: []yacymodel.URLHash{document},
		}},
	)

	if len(calls.paths) != 1 || calls.paths[0] != yacyproto.PathURLMetadata {
		t.Fatalf("the peer was called at %v, want %q", calls.paths, yacyproto.PathURLMetadata)
	}
	if got := calls.forms[0].Get(yacyproto.FieldHashes); got != document.String() {
		t.Errorf("hashes = %q, want the document the ask named", got)
	}
	if got := calls.forms[0].Get(yacyproto.FieldCall); got != yacyproto.CallURLHashList {
		t.Errorf("call = %q, want %q", got, yacyproto.CallURLHashList)
	}
	if got := calls.forms[0].Get(yacyproto.FieldNetworkName); got != networkName {
		t.Errorf("network name = %q, want %q", got, networkName)
	}
}

func TestAPeerThatRejectsTheURLMetadataCallYieldsNoItems(t *testing.T) {
	t.Parallel()

	observer := &recordedOutcome{}
	address, _ := peerAnsweringURLMetadata(
		t,
		`<rss><yacy><response>rejected - insufficient call parameters</response></yacy>`+
			`<channel></channel></rss>`,
	)

	answeredAsks := wireTo(observer).AskForURLMetadata(
		t.Context(),
		[]peerasks.URLMetadataAsk{{
			Peer:      peerAt(address),
			Documents: []yacymodel.URLHash{mustParseURLHash(t, "Q_ylfl--9bK5")},
		}},
	)

	if len(answeredAsks) != 0 {
		t.Fatalf("AskForURLMetadata = %+v, want nothing from a peer that rejected it", answeredAsks)
	}
	if observer.askedFor != peerasks.URLMetadata {
		t.Errorf(
			"the failure was reported for %q, want %q",
			observer.askedFor,
			peerasks.URLMetadata,
		)
	}
}

func TestASearchDocumentsAskNamesTheDocumentsToMatch(t *testing.T) {
	t.Parallel()

	word := yacymodel.WordHash("berlin")
	documents := []yacymodel.URLHash{
		mustParseURLHash(t, "bbbbbbAAAAAA"),
		mustParseURLHash(t, "Q_ylfl--9bK5"),
	}
	address, calls := peerRecordingTheFormPosted(t, "")

	wireTo(&recordedOutcome{}).AskForSearchDocuments(
		callWithin(t, spreadBudgetOfTheTests),
		[]peerasks.SearchDocumentsAsk{
			{Peer: peerAt(address), Word: word, DocumentsToMatch: documents},
		},
	)

	if len(calls.paths) != 1 || calls.paths[0] != yacyproto.PathSearch {
		t.Fatalf("the peer was called at %v, want %q", calls.paths, yacyproto.PathSearch)
	}
	if got := calls.forms[0].Get(
		yacyproto.FieldURLs,
	); got != documents[0].String()+documents[1].String() {
		t.Fatalf("urls = %q, want the documents to match", got)
	}
}

func peerRecordingTheFormPosted(t *testing.T, body string) (string, *peerCalls) {
	t.Helper()

	calls := &peerCalls{}
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, reader *http.Request) {
			calls.paths = append(calls.paths, reader.URL.Path)
			if err := reader.ParseForm(); err == nil {
				calls.forms = append(calls.forms, reader.PostForm)
			}
			_, _ = writer.Write([]byte(body))
		},
	))
	t.Cleanup(server.Close)

	return server.URL, calls
}

func TestAPeerCallWaitsForAnInFlightSlotBeforeItTakesOne(t *testing.T) {
	t.Parallel()

	observer := &recordedOutcome{}
	address, _ := peerAnswering(t, searchAnswerHolding(t), http.StatusOK)

	matchedDocumentsOf(t, observer, peerasks.SearchDocumentsAsk{Peer: peerAt(address)})

	if observer.waitedForASlot != 1 || observer.tookASlot != 1 ||
		observer.waitsBeforeTheSlotWasTaken != 1 {
		t.Fatalf(
			"the call reported %d waits and %d taken slots, %d of the waits before a slot was taken,"+
				" want one wait reported before the one slot it took",
			observer.waitedForASlot,
			observer.tookASlot,
			observer.waitsBeforeTheSlotWasTaken,
		)
	}
}

func TestACallCancelledWhileThePeerAnswersIsNotReportedAsAnUnreachablePeer(t *testing.T) {
	t.Parallel()

	observer := &recordedOutcome{}
	askReachedThePeer, searchEnded := make(chan struct{}), make(chan struct{})
	var oneAskReported sync.Once
	server := httptest.NewServer(http.HandlerFunc(
		func(_ http.ResponseWriter, _ *http.Request) {
			oneAskReported.Do(func() { close(askReachedThePeer) })
			<-searchEnded
		},
	))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(searchEnded) })
	ctx, endSearch := context.WithCancel(t.Context())
	t.Cleanup(endSearch)
	go func() {
		<-askReachedThePeer
		endSearch()
	}()

	answeredAsks := wireTo(observer).AskForSearchDocuments(
		ctx, []peerasks.SearchDocumentsAsk{{Peer: peerAt(server.URL)}},
	)

	if len(answeredAsks) != 0 || observer.cancelled != 1 || observer.unreachable != 0 {
		t.Fatalf(
			"AskForSearchDocuments = %+v with %d cancelled and %d unreachable calls,"+
				" want none answered, one cancelled and none unreachable",
			answeredAsks,
			observer.cancelled,
			observer.unreachable,
		)
	}
	if observer.askedFor != peerasks.SearchDocuments {
		t.Fatalf(
			"the cancelled call named %q, want the search documents",
			observer.askedFor,
		)
	}
}

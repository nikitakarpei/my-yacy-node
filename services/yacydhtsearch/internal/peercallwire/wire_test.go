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
	peerCallBudget = 4 * time.Second
)

type recordedOutcome struct {
	mutex                      sync.Mutex
	answeredMatchedItems       int
	amountOfMatchedItems       int
	answeredURLMetadata        int
	amountOfDescribedDocuments int
	answeredHeldDocuments      int
	amountOfDocuments          int
	refused                    int
	unreachable                int
	unreadable                 int
	askedFor                   peerasks.AskedFor
	spent                      time.Duration
}

func (r *recordedOutcome) PeerAnsweredMatchedItems(
	_ context.Context,
	_ string,
	amountOfMatchedItems int,
	spent time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.answeredMatchedItems++
	r.amountOfMatchedItems = amountOfMatchedItems
	r.spent = spent
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

func (r *recordedOutcome) PeerAnsweredHeldDocuments(
	_ context.Context,
	_ string,
	amountOfDocuments int,
	spent time.Duration,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.answeredHeldDocuments++
	r.amountOfDocuments = amountOfDocuments
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

func wireTo(observer peercallwire.PeerCallObserver) peercallwire.Wire {
	return peercallwire.New(
		http.DefaultClient,
		peercallwire.SearchedNetwork{Name: networkName, RingPartitions: ringPartitions},
		responseLimit,
		observer,
	)
}

func callWithin(t *testing.T, budget time.Duration) context.Context {
	t.Helper()

	ctx, endCall := context.WithTimeout(t.Context(), budget)
	t.Cleanup(endCall)

	return ctx
}

func matchedItemsOf(
	t *testing.T,
	observer peercallwire.PeerCallObserver,
	ask peerasks.MatchedItemsAsk,
) ([]peerasks.MatchedDocument, bool) {
	t.Helper()

	answeredAsks := wireTo(observer).AskForMatchedItems(
		callWithin(t, peerCallBudget), []peerasks.MatchedItemsAsk{ask},
	)
	if len(answeredAsks) == 0 {
		return nil, false
	}

	return answeredAsks[0].MatchedDocuments, true
}

func heldDocumentsAnswerOf(
	t *testing.T,
	observer peercallwire.PeerCallObserver,
	ask peerasks.HeldDocumentsAsk,
) (peerasks.AnsweredHeldDocumentsAsk, bool) {
	t.Helper()

	answeredAsks := wireTo(observer).AskForHeldDocuments(
		callWithin(t, peerCallBudget), []peerasks.HeldDocumentsAsk{ask},
	)
	if len(answeredAsks) == 0 {
		return peerasks.AnsweredHeldDocumentsAsk{}, false
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

	return yacyproto.SearchResponse{
		Count:     len(resources),
		Resources: resources,
	}.Encode().Encode()
}

func mustParseURLHash(t *testing.T, raw string) yacymodel.URLHash {
	t.Helper()

	hash, err := yacymodel.ParseURLHash(raw)
	if err != nil {
		t.Fatalf("ParseURLHash(%q): %v", raw, err)
	}

	return hash
}

func TestAPeerAnswerBecomesResultItems(t *testing.T) {
	t.Parallel()

	observer := &recordedOutcome{}
	address, _ := peerAnswering(
		t, searchAnswerHolding(t, "https://example.org/weather"), http.StatusOK,
	)

	items, replied := matchedItemsOf(t, observer, peerasks.MatchedItemsAsk{
		Peer: peerAt(address),
	})

	if !replied || len(items) != 1 || items[0].Metadata.Address != "https://example.org/weather" {
		t.Fatalf(
			"AskForMatchedItems = %+v, %v, want the address the peer reported", items, replied,
		)
	}
	if observer.answeredMatchedItems != 1 || observer.amountOfMatchedItems != 1 {
		t.Fatalf(
			"PeerAnsweredMatchedItems reported %d times with %d items, want once with one",
			observer.answeredMatchedItems,
			observer.amountOfMatchedItems,
		)
	}
	if observer.spent <= 0 {
		t.Fatalf(
			"PeerAnsweredMatchedItems reported %v spent, want the time the call took",
			observer.spent,
		)
	}
}

func TestAMatchedItemsAskCarriesTheQueryAndTheNetworkOfThisNode(t *testing.T) {
	t.Parallel()

	address, requests := peerAnswering(t, searchAnswerHolding(t), http.StatusOK)
	query := searchquery.QueryFrom("berlin -rain")
	query.Language = "de"

	matchedItemsOf(t, &recordedOutcome{}, peerasks.MatchedItemsAsk{
		Peer:          peerAt(address),
		WordsToMatch:  query.TermHashes(),
		ExcludedWords: query.ExclusionHashes(),
		Language:      query.Language,
		ItemsCeiling:  10,
	})

	if len(requests.received) != 1 {
		t.Fatalf("the peer received %d requests, want one", len(requests.received))
	}
	request := requests.received[0]
	if request.NetworkName != networkName || request.Partitions != int(ringPartitions) {
		t.Fatalf("request = %+v, want the network facts of this node", request)
	}
	grantedAnswerTime := time.Duration(request.Time) * time.Millisecond
	if grantedAnswerTime <= 0 || grantedAnswerTime >= peerCallBudget {
		t.Fatalf(
			"the peer was granted %v of the %v the call had, want less than this node waits",
			grantedAnswerTime,
			peerCallBudget,
		)
	}
	if len(request.Query) != 1 || request.Query[0] != yacymodel.WordHash("berlin") ||
		len(request.Exclude) != 1 || request.Language != "de" || request.Count != 10 {
		t.Fatalf("request = %+v, want the query the ask named", request)
	}
}

func TestAHeldDocumentsAskAsksForTheAbstractAndTheItemsOfOneWordOnly(t *testing.T) {
	t.Parallel()

	word := yacymodel.WordHash("berlin")
	document := mustParseURLHash(t, "bbbbbbAAAAAA")
	body := yacyproto.SearchResponse{
		IndexAbstract: map[yacymodel.Hash][]yacymodel.URLHash{word: {document}},
	}.Encode().Encode()
	address, requests := peerAnswering(t, body, http.StatusOK)
	observer := &recordedOutcome{}

	answeredAsk, replied := heldDocumentsAnswerOf(
		t,
		observer,
		peerasks.HeldDocumentsAsk{Peer: peerAt(address), Word: word, ItemsCeiling: 7},
	)

	if !replied || len(answeredAsk.DocumentsHeldForTheWord) != 1 ||
		answeredAsk.DocumentsHeldForTheWord[0] != document {
		t.Fatalf(
			"AskForHeldDocuments = %+v, %v, want the document the peer holds",
			answeredAsk.DocumentsHeldForTheWord,
			replied,
		)
	}
	request := requests.received[0]
	if len(request.Abstracts.Hashes()) != 1 || request.Abstracts.Hashes()[0] != word ||
		len(request.Query) != 1 || request.Query[0] != word || request.Count != 7 {
		t.Fatalf("request = %+v, want the abstract and the items of the one word", request)
	}
	if observer.answeredHeldDocuments != 1 || observer.amountOfDocuments != 1 {
		t.Fatalf(
			"PeerAnsweredHeldDocuments reported %d times with %d documents, want once with one",
			observer.answeredHeldDocuments,
			observer.amountOfDocuments,
		)
	}
}

func TestAHeldDocumentsAnswerReadsTheItemsAndTheDocumentsThePeerHolds(t *testing.T) {
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

	answeredAsk, replied := heldDocumentsAnswerOf(
		t, &recordedOutcome{}, peerasks.HeldDocumentsAsk{Peer: peerAt(address), Word: word},
	)

	if !replied || len(answeredAsk.MatchedDocuments) != 2 ||
		answeredAsk.MatchedDocuments[0].Metadata.Address != "https://example.org/berlin" {
		t.Fatalf("AskForHeldDocuments read %+v, want the documents the peer matched",
			answeredAsk.MatchedDocuments)
	}
	if answeredAsk.MatchedDocuments[0].CountOfAWordTheAskNamed.Hits != 3 ||
		answeredAsk.MatchedDocuments[1].CountOfAWordTheAskNamed.CountedByAPeer() {
		t.Fatalf("the documents carry the counts %+v, want the one the peer counted a word in",
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

	answeredAsk, replied := heldDocumentsAnswerOf(
		t, &recordedOutcome{}, peerasks.HeldDocumentsAsk{Peer: peerAt(address), Word: word},
	)

	if _, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get(); !replied || counted {
		t.Fatalf("the answer carries %+v documents held for the word, want none",
			answeredAsk.AmountOfDocumentsHeldForTheWord)
	}
}

func TestAHeldDocumentsAskReadsNoDocumentOfAnotherWord(t *testing.T) {
	t.Parallel()

	body := yacyproto.SearchResponse{
		IndexAbstract: map[yacymodel.Hash][]yacymodel.URLHash{
			yacymodel.WordHash("weather"): {mustParseURLHash(t, "bbbbbbAAAAAA")},
		},
	}.Encode().Encode()
	address, _ := peerAnswering(t, body, http.StatusOK)

	answeredAsk, replied := heldDocumentsAnswerOf(
		t,
		&recordedOutcome{},
		peerasks.HeldDocumentsAsk{Peer: peerAt(address), Word: yacymodel.WordHash("berlin")},
	)

	if !replied || len(answeredAsk.DocumentsHeldForTheWord) != 0 {
		t.Fatalf("AskForHeldDocuments = %+v, want no document of the word it did not ask for",
			answeredAsk.DocumentsHeldForTheWord)
	}
}

func TestAPeerThatRefusesTheSearchYieldsNoItems(t *testing.T) {
	t.Parallel()

	observer := &recordedOutcome{}
	address, _ := peerAnswering(t, "", http.StatusServiceUnavailable)

	items, replied := matchedItemsOf(t, observer, peerasks.MatchedItemsAsk{Peer: peerAt(address)})

	if replied || len(items) != 0 || observer.refused != 1 {
		t.Fatalf(
			"AskForMatchedItems = %+v with %d refusals, want none and one", items, observer.refused,
		)
	}
}

func TestAFailedCallReportsWhatItAskedThePeerFor(t *testing.T) {
	t.Parallel()

	address, _ := peerAnswering(t, "", http.StatusServiceUnavailable)

	refusedItemsAsk := &recordedOutcome{}
	matchedItemsOf(t, refusedItemsAsk, peerasks.MatchedItemsAsk{Peer: peerAt(address)})

	refusedDocumentsAsk := &recordedOutcome{}
	heldDocumentsAnswerOf(t, refusedDocumentsAsk, peerasks.HeldDocumentsAsk{
		Peer: peerAt(address),
		Word: yacymodel.WordHash("berlin"),
	})

	if refusedItemsAsk.askedFor != peerasks.MatchedItems ||
		refusedDocumentsAsk.askedFor != peerasks.HeldDocuments {
		t.Fatalf(
			"the refusals named %q and %q, want the matched items and the held documents",
			refusedItemsAsk.askedFor,
			refusedDocumentsAsk.askedFor,
		)
	}
}

func TestAPeerThatHoldsNothingStillReplies(t *testing.T) {
	t.Parallel()

	address, _ := peerAnswering(t, searchAnswerHolding(t), http.StatusOK)

	items, replied := matchedItemsOf(
		t, &recordedOutcome{}, peerasks.MatchedItemsAsk{Peer: peerAt(address)},
	)

	if !replied || len(items) != 0 {
		t.Fatalf("AskForMatchedItems = %+v, %v, want a reply that carries nothing", items, replied)
	}
}

func TestAPeerThatCannotBeReachedYieldsNoItems(t *testing.T) {
	t.Parallel()

	observer := &recordedOutcome{}

	items, replied := matchedItemsOf(
		t, observer, peerasks.MatchedItemsAsk{Peer: peerAt("http://127.0.0.1:1")},
	)

	if replied || len(items) != 0 || observer.unreachable != 1 {
		t.Fatalf(
			"AskForMatchedItems = %+v with %d unreachable, want none and one",
			items,
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
	if observer.answeredMatchedItems != 0 {
		t.Fatal("a metadata answer was reported as items a peer matched")
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

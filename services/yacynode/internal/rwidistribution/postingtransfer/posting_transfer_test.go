package postingtransfer_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingtransfer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/urlmetadatacourier"
)

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}

type fakePostingCourier struct {
	receipt postingcourier.Receipt
}

func (f fakePostingCourier) Offer(
	context.Context,
	yacymodel.Seed,
	[]yacymodel.RWIPosting,
) postingcourier.Receipt {
	return f.receipt
}

type fakeURLMetadataCourier struct {
	receipt   urlmetadatacourier.Receipt
	delivered []yacymodel.URLMetadata
}

func (f *fakeURLMetadataCourier) Deliver(
	_ context.Context,
	_ yacymodel.Seed,
	metadata []yacymodel.URLMetadata,
) urlmetadatacourier.Receipt {
	f.delivered = append(f.delivered, metadata...)

	return f.receipt
}

type fakeURLDirectory struct {
	metadata map[yacymodel.URLHash]yacymodel.URLMetadata
	unread   error
}

func (f fakeURLDirectory) MetadataPerHash(
	_ *vault.Txn,
	hashes []yacymodel.URLHash,
) (map[yacymodel.URLHash]yacymodel.URLMetadata, error) {
	if f.unread != nil {
		return nil, f.unread
	}

	found := make(map[yacymodel.URLHash]yacymodel.URLMetadata, len(hashes))
	for _, hash := range hashes {
		if entry, ok := f.metadata[hash]; ok {
			found[hash] = entry
		}
	}

	return found, nil
}

type fakeObserver struct {
	offers                map[string]int
	postingsOffered       map[string]int
	urlMetadataDeliveries map[string]int
	urlsDelivered         map[string]int
	urlsUnknownToUs       int
}

func newFakeObserver() *fakeObserver {
	return &fakeObserver{
		offers:                make(map[string]int),
		postingsOffered:       make(map[string]int),
		urlMetadataDeliveries: make(map[string]int),
		urlsDelivered:         make(map[string]int),
	}
}

func (f *fakeObserver) ObservePostingOffer(outcome string, postings int) {
	f.offers[outcome]++
	f.postingsOffered[outcome] += postings
}

func (f *fakeObserver) ObserveURLMetadataDelivery(outcome string, urls int) {
	f.urlMetadataDeliveries[outcome]++
	f.urlsDelivered[outcome] += urls
}

func (f *fakeObserver) ObserveURLsUnknownToUs(urls int) {
	f.urlsUnknownToUs += urls
}

func seed(hash yacymodel.Hash) yacymodel.Seed {
	return yacymodel.Seed{Hash: hash}
}

func fakePosting(word yacymodel.Hash, url yacymodel.URLHash) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{WordHash: word, URLHash: url}
}

func TestSendAcceptsPostingsWithNoUnknownURLs(t *testing.T) {
	peer := yacymodel.WordHash("peer")
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	posting := fakePosting(word, url)
	observer := newFakeObserver()
	postingTransfers := postingtransfer.New(
		openVault(t), fakePostingCourier{
			receipt: postingcourier.Receipt{Outcome: postingcourier.Accepted},
		},
		&fakeURLMetadataCourier{},
		fakeURLDirectory{},
		observer,
	)

	answer := postingTransfers.Send(
		context.Background(), seed(peer), []yacymodel.RWIPosting{posting},
	)

	if answer.Result != postingtransfer.OfferAccepted || len(answer.AcceptedPostings) != 1 {
		t.Fatalf("answer = %+v, want the posting accepted", answer)
	}
}

func TestSendExcludesPostingWhenURLMetadataDeliveryFails(t *testing.T) {
	peer := yacymodel.WordHash("peer")
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	observer := newFakeObserver()
	postingTransfers := postingtransfer.New(
		openVault(t), fakePostingCourier{receipt: postingcourier.Receipt{
			Outcome:           postingcourier.Accepted,
			URLsUnknownToPeer: []yacymodel.URLHash{url},
		}},
		&fakeURLMetadataCourier{
			receipt: urlmetadatacourier.Receipt{Outcome: urlmetadatacourier.Deferred},
		},
		fakeURLDirectory{
			metadata: map[yacymodel.URLHash]yacymodel.URLMetadata{
				url: {Address: "http://example.com/u1"},
			},
		},
		observer,
	)

	answer := postingTransfers.Send(
		context.Background(), seed(peer), []yacymodel.RWIPosting{fakePosting(word, url)},
	)

	if len(answer.AcceptedPostings) != 0 {
		t.Fatalf(
			"AcceptedPostings = %v, want none when url metadata delivery fails",
			answer.AcceptedPostings,
		)
	}
	if observer.urlMetadataDeliveries[string(urlmetadatacourier.Deferred)] != 1 {
		t.Fatalf(
			"observed url metadata deliveries = %+v, want 1 for outcome %q",
			observer.urlMetadataDeliveries, urlmetadatacourier.Deferred,
		)
	}
}

func TestSendDeliversMetadataItHasWhenOneURLIsAbsent(t *testing.T) {
	peer := yacymodel.WordHash("peer")
	word := yacymodel.WordHash("w1")
	present, absent := urlHash("u1"), urlHash("u2")
	metadataCourier := &fakeURLMetadataCourier{
		receipt: urlmetadatacourier.Receipt{Outcome: urlmetadatacourier.Accepted},
	}
	observer := newFakeObserver()
	postingTransfers := postingtransfer.New(
		openVault(t), fakePostingCourier{receipt: postingcourier.Receipt{
			Outcome:           postingcourier.Accepted,
			URLsUnknownToPeer: []yacymodel.URLHash{present, absent},
		}},
		metadataCourier,
		fakeURLDirectory{
			metadata: map[yacymodel.URLHash]yacymodel.URLMetadata{
				present: {Address: "http://example.com/u1"},
			},
		},
		observer,
	)

	answer := postingTransfers.Send(
		context.Background(),
		seed(peer),
		[]yacymodel.RWIPosting{fakePosting(word, present), fakePosting(word, absent)},
	)

	if len(metadataCourier.delivered) != 1 {
		t.Fatalf(
			"delivered = %v, want the one url whose metadata this node holds",
			metadataCourier.delivered,
		)
	}
	if observer.urlsUnknownToUs != 1 {
		t.Fatalf(
			"observed urls unknown to us = %d, want the one url whose metadata this node lacks",
			observer.urlsUnknownToUs,
		)
	}
	if len(answer.AcceptedPostings) != 1 || answer.AcceptedPostings[0].URLHash != present {
		t.Fatalf(
			"AcceptedPostings = %v, want only the posting with delivered metadata",
			answer.AcceptedPostings,
		)
	}
}

func openVault(t *testing.T) *vault.Vault {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = v.Close() })

	return v
}

func TestSendFailsOnlyWhenThePeerGivesNoAnswer(t *testing.T) {
	resultPerOutcome := map[postingcourier.Outcome]postingtransfer.OfferResult{
		postingcourier.Unreachable:   postingtransfer.OfferFailed,
		postingcourier.Unaddressable: postingtransfer.OfferFailed,
		postingcourier.Deferred:      postingtransfer.OfferDeclined,
		postingcourier.Overloaded:    postingtransfer.OfferDeclined,
		postingcourier.Refused:       postingtransfer.OfferDeclined,
		postingcourier.Accepted:      postingtransfer.OfferAccepted,
	}
	for outcome, wantResult := range resultPerOutcome {
		t.Run(string(outcome), func(t *testing.T) {
			postingTransfers := postingtransfer.New(
				openVault(t),
				fakePostingCourier{receipt: postingcourier.Receipt{Outcome: outcome}},
				&fakeURLMetadataCourier{},
				fakeURLDirectory{},
				newFakeObserver(),
			)

			answer := postingTransfers.Send(
				context.Background(),
				seed(yacymodel.WordHash("peer")),
				[]yacymodel.RWIPosting{fakePosting(yacymodel.WordHash("w1"), urlHash("u1"))},
			)

			if answer.Result != wantResult {
				t.Fatalf("Result = %v, want %v", answer.Result, wantResult)
			}
		})
	}
}

func TestADeclinedOfferCarriesThePausePeerRequested(t *testing.T) {
	requestedPause := 30 * time.Second
	postingTransfers := postingtransfer.New(
		openVault(t),
		fakePostingCourier{receipt: postingcourier.Receipt{
			Outcome:        postingcourier.Deferred,
			RequestedPause: requestedPause,
		}},
		&fakeURLMetadataCourier{},
		fakeURLDirectory{},
		newFakeObserver(),
	)

	answer := postingTransfers.Send(
		context.Background(),
		seed(yacymodel.WordHash("peer")),
		[]yacymodel.RWIPosting{fakePosting(yacymodel.WordHash("w1"), urlHash("u1"))},
	)

	if answer.Result != postingtransfer.OfferDeclined || answer.RequestedPause != requestedPause {
		t.Fatalf("answer = %+v, want declined with pause %v", answer, requestedPause)
	}
}

func TestAnOverloadedOfferCarriesThePausePeerRequested(t *testing.T) {
	postingTransfers := postingtransfer.New(
		openVault(t),
		fakePostingCourier{receipt: postingcourier.Receipt{
			Outcome:        postingcourier.Overloaded,
			RequestedPause: time.Minute,
		}},
		&fakeURLMetadataCourier{},
		fakeURLDirectory{},
		newFakeObserver(),
	)

	answer := postingTransfers.Send(
		context.Background(),
		seed(yacymodel.WordHash("peer")),
		[]yacymodel.RWIPosting{fakePosting(yacymodel.WordHash("w1"), urlHash("u1"))},
	)

	if answer.Result != postingtransfer.OfferDeclined || answer.RequestedPause != time.Minute {
		t.Fatalf("answer = %+v, want declined with pause 1m", answer)
	}
}

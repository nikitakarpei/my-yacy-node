package postingtransfer_test

import (
	"context"
	"testing"

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
	string,
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
	_ string,
	_ yacymodel.Hash,
	metadata []yacymodel.URLMetadata,
) urlmetadatacourier.Receipt {
	f.delivered = append(f.delivered, metadata...)

	return f.receipt
}

type fakeURLDirectory struct {
	metadata map[yacymodel.URLHash]yacymodel.URLMetadata
	unread   error
}

func (f fakeURLDirectory) MetadataByHash(
	_ *vault.Txn,
	hashes []yacymodel.URLHash,
) ([]yacymodel.URLMetadata, error) {
	if f.unread != nil {
		return nil, f.unread
	}

	found := make([]yacymodel.URLMetadata, 0, len(hashes))
	for _, hash := range hashes {
		if entry, ok := f.metadata[hash]; ok {
			found = append(found, entry)
		}
	}

	return found, nil
}

func (f fakeURLDirectory) MissingURLs(
	_ *vault.Txn,
	hashes []yacymodel.URLHash,
) ([]yacymodel.URLHash, error) {
	missing := make([]yacymodel.URLHash, 0, len(hashes))
	for _, hash := range hashes {
		if _, ok := f.metadata[hash]; !ok {
			missing = append(missing, hash)
		}
	}

	return missing, nil
}

func (fakeURLDirectory) Count(*vault.Txn) (int, error) { return 0, nil }

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

var reachableHost = func() yacymodel.Host {
	host, err := yacymodel.ParseHost("127.0.0.1")
	if err != nil {
		panic(err)
	}

	return host
}()

var reachablePort = func() yacymodel.Port {
	port, err := yacymodel.ParsePort("8090")
	if err != nil {
		panic(err)
	}

	return port
}()

func seed(hash yacymodel.Hash) yacymodel.Seed {
	return yacymodel.Seed{
		Hash:           hash,
		PrimaryAddress: yacymodel.Some(reachableHost),
		Port:           yacymodel.Some(reachablePort),
		Capabilities:   yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: true}),
	}
}

func fakePosting(word yacymodel.Hash, url yacymodel.URLHash) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{WordHash: word, URLHash: url}
}

func unaddressableSeed(hash yacymodel.Hash) yacymodel.Seed {
	return yacymodel.Seed{
		Hash:         hash,
		Capabilities: yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: true}),
	}
}

func TestSendReportsUnaddressablePeer(t *testing.T) {
	peer := yacymodel.WordHash("peer")
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	observer := newFakeObserver()
	postingTransfers := postingtransfer.New(
		openVault(t), fakePostingCourier{}, &fakeURLMetadataCourier{}, fakeURLDirectory{}, observer,
	)

	answer := postingTransfers.Send(
		context.Background(),
		unaddressableSeed(peer),
		[]yacymodel.RWIPosting{fakePosting(word, url)},
	)

	if answer.Accepted {
		t.Fatalf("Accepted = true, want false for an unaddressable peer")
	}
	if observer.postingsOffered[string(postingcourier.Unaddressable)] != 1 {
		t.Fatalf(
			"observed offers = %+v, want 1 posting for outcome %q",
			observer.postingsOffered, postingcourier.Unaddressable,
		)
	}
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

	if !answer.Accepted || len(answer.AcceptedPostings) != 1 {
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

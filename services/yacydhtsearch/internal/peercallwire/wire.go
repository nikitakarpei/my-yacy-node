// Package peercallwire speaks the YaCy search protocol to the peers an ask
// names. It asks a peer for the items it matches, for the documents it holds
// for one word together with the items it puts first for that word, or for the
// metadata it holds for documents the ask names, reads back what the peer
// answered, and leaves every peer the time the peer
// call has left, less the margin the answer needs to reach this node; a peer
// call with no deadline leaves the peer a time of its own. It puts every ask of
// one round at once, because the spread that names the asks holds their amount
// within the peer calls one query may put. It carries the facts that hold for
// every peer call of this node — the network it searches and the partitions of
// the ring.
package peercallwire

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const grantedAnswerMargin = time.Second

type peerCall struct {
	address  string
	path     string
	askedFor peerasks.AskedFor
	form     url.Values
}

type Wire struct {
	client           *http.Client
	searchedNetwork  SearchedNetwork
	maxResponseBytes int64
	observer         PeerCallObserver
}

func New(
	client *http.Client,
	searchedNetwork SearchedNetwork,
	maxResponseBytes int64,
	observer PeerCallObserver,
) Wire {
	return Wire{
		client:           client,
		searchedNetwork:  searchedNetwork,
		maxResponseBytes: maxResponseBytes,
		observer:         observer,
	}
}

func (w Wire) AskForMatchedItems(
	ctx context.Context,
	asks []peerasks.MatchedItemsAsk,
) []peerasks.AnsweredMatchedItemsAsk {
	return putAsksToPeers(
		asks,
		func(ask peerasks.MatchedItemsAsk) (peerasks.AnsweredMatchedItemsAsk, bool) {
			return w.putMatchedItemsAsk(ctx, ask)
		},
	)
}

func putAsksToPeers[Ask any, Answered any](
	asks []Ask,
	putAsk func(Ask) (Answered, bool),
) []Answered {
	answeredAsks := make([]Answered, len(asks))
	replied := make([]bool, len(asks))
	var calls sync.WaitGroup

	for index, ask := range asks {
		calls.Add(1)
		go func() {
			defer calls.Done()
			answeredAsks[index], replied[index] = putAsk(ask)
		}()
	}
	calls.Wait()

	return answeredAsksThatCameBack(answeredAsks, replied)
}

func answeredAsksThatCameBack[Answered any](
	answeredAsks []Answered,
	replied []bool,
) []Answered {
	keptAnsweredAsks := make([]Answered, 0, len(answeredAsks))
	for index, answeredAsk := range answeredAsks {
		if !replied[index] {
			continue
		}
		keptAnsweredAsks = append(keptAnsweredAsks, answeredAsk)
	}

	return keptAnsweredAsks
}

func (w Wire) putMatchedItemsAsk(
	ctx context.Context,
	ask peerasks.MatchedItemsAsk,
) (peerasks.AnsweredMatchedItemsAsk, bool) {
	startedAt := time.Now()
	response, ok := w.searchResponse(
		ctx,
		peerCall{
			address:  ask.Peer.Address,
			path:     yacyproto.PathSearch,
			askedFor: peerasks.MatchedItems,
			form:     w.requestForMatchedItems(ctx, ask).Form(),
		},
		startedAt,
	)
	if !ok {
		return peerasks.AnsweredMatchedItemsAsk{}, false
	}

	items := itemsOf(response)
	w.observer.PeerAnsweredMatchedItems(ctx, ask.Peer.Address, len(items), time.Since(startedAt))

	return peerasks.AnsweredMatchedItemsAsk{Ask: ask, Items: items}, true
}

func (w Wire) requestForMatchedItems(
	ctx context.Context,
	ask peerasks.MatchedItemsAsk,
) yacyproto.SearchRequest {
	request := w.requestFor(ctx, ask.ExcludedWords, ask.Language)
	request.Query = ask.WordsToMatch
	request.Count = ask.ItemsCeiling

	return request
}

func (w Wire) requestFor(
	ctx context.Context,
	excludedWords []yacymodel.Hash,
	language string,
) yacyproto.SearchRequest {
	return yacyproto.SearchRequest{
		NetworkName: w.searchedNetwork.Name,
		Exclude:     excludedWords,
		Time:        grantedAnswerTimeOf(ctx),
		Partitions:  int(w.searchedNetwork.RingPartitions),
		ContentDom:  yacyproto.ContentDomainText,
		Language:    language,
	}
}

func grantedAnswerTimeOf(ctx context.Context) int {
	deadline, bounded := ctx.Deadline()
	if !bounded {
		return 0
	}

	return int(max(0, time.Until(deadline)-grantedAnswerMargin).Milliseconds())
}

func itemsOf(response yacyproto.SearchResponse) []searchresult.Item {
	items := make([]searchresult.Item, 0, len(response.Resources))
	for _, resource := range response.Resources {
		items = append(items, searchresult.ItemFrom(resource.Metadata))
	}

	return items
}

func (w Wire) AskForURLMetadata(
	ctx context.Context,
	asks []peerasks.URLMetadataAsk,
) []peerasks.AnsweredURLMetadataAsk {
	return putAsksToPeers(
		asks,
		func(ask peerasks.URLMetadataAsk) (peerasks.AnsweredURLMetadataAsk, bool) {
			return w.putURLMetadataAsk(ctx, ask)
		},
	)
}

func (w Wire) putURLMetadataAsk(
	ctx context.Context,
	ask peerasks.URLMetadataAsk,
) (peerasks.AnsweredURLMetadataAsk, bool) {
	startedAt := time.Now()
	response, ok := w.urlMetadataResponse(
		ctx,
		peerCall{
			address:  ask.Peer.Address,
			path:     yacyproto.PathURLMetadata,
			askedFor: peerasks.URLMetadata,
			form:     w.requestForURLMetadata(ask).Form(),
		},
		startedAt,
	)
	if !ok {
		return peerasks.AnsweredURLMetadataAsk{}, false
	}

	items := itemsOfURLMetadata(response.URLs)
	w.observer.PeerAnsweredURLMetadata(ctx, ask.Peer.Address, len(items), time.Since(startedAt))

	return peerasks.AnsweredURLMetadataAsk{Ask: ask, Items: items}, true
}

func (w Wire) requestForURLMetadata(ask peerasks.URLMetadataAsk) yacyproto.URLMetadataRequest {
	return yacyproto.URLMetadataRequest{
		NetworkName: w.searchedNetwork.Name,
		URLs:        ask.Documents,
	}
}

func (w Wire) urlMetadataResponse(
	ctx context.Context,
	call peerCall,
	startedAt time.Time,
) (yacyproto.URLMetadataResponse, bool) {
	body, ok := w.answerBody(ctx, call, startedAt)
	if !ok {
		return yacyproto.URLMetadataResponse{}, false
	}

	response, err := yacyproto.ParseURLMetadataResponse(ctx, []byte(body))
	if err != nil {
		w.observer.PeerAnswerUnreadable(
			ctx, call.address, call.askedFor, err, time.Since(startedAt),
		)

		return yacyproto.URLMetadataResponse{}, false
	}

	return response, true
}

func itemsOfURLMetadata(urls []yacymodel.URLMetadata) []searchresult.Item {
	items := make([]searchresult.Item, 0, len(urls))
	for _, metadata := range urls {
		items = append(items, searchresult.ItemFrom(metadata))
	}

	return items
}

func (w Wire) AskForHeldDocuments(
	ctx context.Context,
	asks []peerasks.HeldDocumentsAsk,
) []peerasks.AnsweredHeldDocumentsAsk {
	return putAsksToPeers(
		asks,
		func(ask peerasks.HeldDocumentsAsk) (peerasks.AnsweredHeldDocumentsAsk, bool) {
			return w.putHeldDocumentsAsk(ctx, ask)
		},
	)
}

func (w Wire) putHeldDocumentsAsk(
	ctx context.Context,
	ask peerasks.HeldDocumentsAsk,
) (peerasks.AnsweredHeldDocumentsAsk, bool) {
	startedAt := time.Now()
	response, ok := w.searchResponse(
		ctx,
		peerCall{
			address:  ask.Peer.Address,
			path:     yacyproto.PathSearch,
			askedFor: peerasks.HeldDocuments,
			form:     w.requestForHeldDocuments(ctx, ask).Form(),
		},
		startedAt,
	)
	if !ok {
		return peerasks.AnsweredHeldDocumentsAsk{}, false
	}

	documents := response.IndexAbstract[ask.Word]
	w.observer.PeerAnsweredHeldDocuments(
		ctx, ask.Peer.Address, len(documents), time.Since(startedAt),
	)

	return peerasks.AnsweredHeldDocumentsAsk{
		Ask:                             ask,
		Documents:                       documents,
		Items:                           itemsOf(response),
		AmountOfItemsWithAPosting:       amountOfResourcesWithAPosting(response),
		AmountOfDocumentsHeldForTheWord: response.IndexCount[ask.Word],
	}, true
}

func (w Wire) requestForHeldDocuments(
	ctx context.Context,
	ask peerasks.HeldDocumentsAsk,
) yacyproto.SearchRequest {
	request := w.requestFor(ctx, ask.ExcludedWords, ask.Language)
	request.Abstracts = yacyproto.SearchAbstractsOf([]yacymodel.Hash{ask.Word})
	request.Query = []yacymodel.Hash{ask.Word}
	request.Count = ask.ItemsCeiling

	return request
}

func amountOfResourcesWithAPosting(response yacyproto.SearchResponse) int {
	amount := 0
	for _, resource := range response.Resources {
		if _, reported := resource.Posting.Get(); !reported {
			continue
		}
		amount++
	}

	return amount
}

func (w Wire) searchResponse(
	ctx context.Context,
	call peerCall,
	startedAt time.Time,
) (yacyproto.SearchResponse, bool) {
	body, ok := w.answerBody(ctx, call, startedAt)
	if !ok {
		return yacyproto.SearchResponse{}, false
	}

	response, err := yacyproto.ParseSearchResponse(ctx, yacyproto.ParseMessage(body))
	if err != nil {
		w.observer.PeerAnswerUnreadable(
			ctx, call.address, call.askedFor, err, time.Since(startedAt),
		)

		return yacyproto.SearchResponse{}, false
	}

	return response, true
}

func (w Wire) answerBody(
	ctx context.Context,
	call peerCall,
	startedAt time.Time,
) (string, bool) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, call.address+call.path, strings.NewReader(call.form.Encode()),
	)
	if err != nil {
		w.observer.PeerUnreachable(ctx, call.address, call.askedFor, err, time.Since(startedAt))
		return "", false
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := w.client.Do(req)
	if err != nil {
		w.observer.PeerUnreachable(ctx, call.address, call.askedFor, err, time.Since(startedAt))
		return "", false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		w.observer.PeerRefused(
			ctx, call.address, call.askedFor, resp.StatusCode, time.Since(startedAt),
		)
		return "", false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, w.maxResponseBytes))
	if err != nil {
		w.observer.PeerAnswerUnreadable(
			ctx, call.address, call.askedFor, err, time.Since(startedAt),
		)
		return "", false
	}

	return string(body), true
}

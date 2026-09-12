// Package peercallwire speaks the YaCy search protocol to the peers an ask
// names. It asks a peer for the documents it matches, for the documents it
// holds for one word, or for the metadata of the documents the ask names, and
// reads back what the peer answered. It holds the peer calls this node has in
// flight at the amount it is built for and puts the asks in the order they
// came. One peer call runs for the budget it is built for, or for the time the
// ask has left, whichever ends first, and leaves the peer that time less the
// margin the answer needs to reach this node.
package peercallwire

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
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
	callsInFlight    peerCallsInFlight
	peerCallBudget   time.Duration
	observer         PeerCallObserver
}

func New(
	client *http.Client,
	searchedNetwork SearchedNetwork,
	limits PeerCallLimits,
	observer PeerCallObserver,
) Wire {
	return Wire{
		client:           client,
		searchedNetwork:  searchedNetwork,
		maxResponseBytes: limits.MaxResponseBytes,
		callsInFlight:    make(peerCallsInFlight, limits.PeerCallsInFlight),
		peerCallBudget:   limits.PeerCallBudget,
		observer:         observer,
	}
}

func (w Wire) AskForMatchedItems(
	ctx context.Context,
	asks []peerasks.MatchedItemsAsk,
) []peerasks.AnsweredMatchedItemsAsk {
	return putAsksToPeers(
		w.callsInFlight,
		asks,
		func(ask peerasks.MatchedItemsAsk) (peerasks.AnsweredMatchedItemsAsk, bool) {
			return w.putMatchedItemsAsk(ctx, ask)
		},
	)
}

func putAsksToPeers[Ask any, Answered any](
	callsInFlight peerCallsInFlight,
	asks []Ask,
	putAsk func(Ask) (Answered, bool),
) []Answered {
	answeredAsks := make([]Answered, len(asks))
	replied := make([]bool, len(asks))

	callsInFlight.putEveryPeerCallInTheOrderGiven(len(asks), func(index int) {
		answeredAsks[index], replied[index] = putAsk(asks[index])
	})

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
	ctx, endPeerCall := context.WithTimeout(ctx, w.peerCallBudget)
	defer endPeerCall()
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

	matchedDocuments := matchedDocumentsOf(response)
	w.observer.PeerAnsweredMatchedItems(
		ctx, ask.Peer.Address, len(matchedDocuments), time.Since(startedAt),
	)

	return peerasks.AnsweredMatchedItemsAsk{Ask: ask, MatchedDocuments: matchedDocuments}, true
}

func matchedDocumentsOf(response yacyproto.SearchResponse) []peerasks.MatchedDocument {
	matchedDocuments := make([]peerasks.MatchedDocument, 0, len(response.Resources))
	for _, resource := range response.Resources {
		matchedDocuments = append(matchedDocuments, peerasks.MatchedDocument{
			Metadata:                resource.Metadata,
			CountOfAWordTheAskNamed: wordCountOf(resource.Posting),
		})
	}

	return matchedDocuments
}

func wordCountOf(posting yacymodel.Optional[yacymodel.RWIPosting]) peeranswers.WordCount {
	counted, sent := posting.Get()
	if !sent {
		return peeranswers.WordCount{}
	}

	return peeranswers.WordCount{Hits: counted.Hits, TextWords: counted.TextWords}
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

func (w Wire) AskForURLMetadata(
	ctx context.Context,
	asks []peerasks.URLMetadataAsk,
) []peerasks.AnsweredURLMetadataAsk {
	return putAsksToPeers(
		w.callsInFlight,
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
	ctx, endPeerCall := context.WithTimeout(ctx, w.peerCallBudget)
	defer endPeerCall()
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

	w.observer.PeerAnsweredURLMetadata(
		ctx, ask.Peer.Address, len(response.URLs), time.Since(startedAt),
	)

	return peerasks.AnsweredURLMetadataAsk{Ask: ask, MetadataOfEachDocument: response.URLs}, true
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

func (w Wire) AskForHeldDocuments(
	ctx context.Context,
	asks []peerasks.HeldDocumentsAsk,
) []peerasks.AnsweredHeldDocumentsAsk {
	return putAsksToPeers(
		w.callsInFlight,
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
	ctx, endPeerCall := context.WithTimeout(ctx, w.peerCallBudget)
	defer endPeerCall()
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

	documentsHeldForTheWord := response.IndexAbstract[ask.Word]
	w.observer.PeerAnsweredHeldDocuments(
		ctx, ask.Peer.Address, len(documentsHeldForTheWord), time.Since(startedAt),
	)

	return peerasks.AnsweredHeldDocumentsAsk{
		Ask:                             ask,
		DocumentsHeldForTheWord:         documentsHeldForTheWord,
		MatchedDocuments:                matchedDocumentsOf(response),
		AmountOfDocumentsHeldForTheWord: amountOfDocumentsHeldForTheWordOf(response, ask.Word),
	}, true
}

func amountOfDocumentsHeldForTheWordOf(
	response yacyproto.SearchResponse,
	word yacymodel.Hash,
) yacymodel.Optional[int] {
	documentsHeld, counted := response.IndexCount[word]
	if !counted {
		return yacymodel.None[int]()
	}

	return yacymodel.Some(documentsHeld)
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

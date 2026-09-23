// Package peercallwire puts asks to peers over the YaCy search protocol and
// reads back what each peer answered.
package peercallwire

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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
		callsInFlight:    newPeerCallsInFlight(limits.PeerCallsInFlight, observer),
		peerCallBudget:   limits.PeerCallBudget,
		observer:         observer,
	}
}

func putAsksToPeers[Ask any, Answered any](
	ctx context.Context,
	callsInFlight peerCallsInFlight,
	asks []Ask,
	askedPeerOf func(Ask) (address string, askedFor peerasks.AskedFor),
	putAsk func(Ask) (Answered, bool),
) []Answered {
	answeredAsks := make([]Answered, len(asks))
	replied := make([]bool, len(asks))

	callsInFlight.putEveryPeerCallInTheOrderGiven(
		ctx,
		len(asks),
		func(index int) (string, peerasks.AskedFor) { return askedPeerOf(asks[index]) },
		func(index int) { answeredAsks[index], replied[index] = putAsk(asks[index]) },
	)

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

func matchedDocumentsOf(response yacyproto.SearchResponse) []peerasks.MatchedDocument {
	matchedDocuments := make([]peerasks.MatchedDocument, 0, len(response.Resources))
	for _, resource := range response.Resources {
		matchedDocuments = append(matchedDocuments, peerasks.MatchedDocument{
			Metadata: resource.Metadata,
			Posting:  resource.Posting,
		})
	}

	return matchedDocuments
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
		ctx,
		w.callsInFlight,
		asks,
		func(ask peerasks.URLMetadataAsk) (string, peerasks.AskedFor) {
			return ask.Peer.Address, peerasks.URLMetadata
		},
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

func (w Wire) AskForSearchDocuments(
	ctx context.Context,
	asks []peerasks.SearchDocumentsAsk,
) []peerasks.AnsweredSearchDocumentsAsk {
	return putAsksToPeers(
		ctx,
		w.callsInFlight,
		asks,
		func(ask peerasks.SearchDocumentsAsk) (string, peerasks.AskedFor) {
			return ask.Peer.Address, peerasks.SearchDocuments
		},
		func(ask peerasks.SearchDocumentsAsk) (peerasks.AnsweredSearchDocumentsAsk, bool) {
			return w.putSearchDocumentsAsk(ctx, ask)
		},
	)
}

func (w Wire) putSearchDocumentsAsk(
	ctx context.Context,
	ask peerasks.SearchDocumentsAsk,
) (peerasks.AnsweredSearchDocumentsAsk, bool) {
	ctx, endPeerCall := context.WithTimeout(ctx, w.peerCallBudget)
	defer endPeerCall()
	startedAt := time.Now()
	response, ok := w.searchResponse(
		ctx,
		peerCall{
			address:  ask.Peer.Address,
			path:     yacyproto.PathSearch,
			askedFor: peerasks.SearchDocuments,
			form:     w.requestForSearchDocuments(ctx, ask).Form(),
		},
		startedAt,
	)
	if !ok {
		return peerasks.AnsweredSearchDocumentsAsk{}, false
	}

	documentsListedForTheWord := response.IndexAbstract[ask.Word]
	matchedDocuments := matchedDocumentsOf(response)
	w.observer.PeerSearchedDocuments(
		ctx,
		ask.Peer.Address,
		len(documentsListedForTheWord),
		len(matchedDocuments),
		time.Since(startedAt),
	)

	return peerasks.AnsweredSearchDocumentsAsk{
		Ask:                             ask,
		DocumentsListedForTheWord:       documentsListedForTheWord,
		MatchedDocuments:                matchedDocuments,
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

func (w Wire) requestForSearchDocuments(
	ctx context.Context,
	ask peerasks.SearchDocumentsAsk,
) yacyproto.SearchRequest {
	request := w.requestFor(ctx, ask.ExcludedWords, ask.Language)
	request.Abstracts = yacyproto.SearchAbstractsOf([]yacymodel.Hash{ask.Word})
	request.Query = append([]yacymodel.Hash{ask.Word}, ask.OtherWordsToMatch...)
	request.URLs = ask.DocumentsToMatch
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
		w.reportUnansweredPeerCall(ctx, call, err, time.Since(startedAt))
		return "", false
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := w.client.Do(req)
	if err != nil {
		w.reportUnansweredPeerCall(ctx, call, err, time.Since(startedAt))
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

func (w Wire) reportUnansweredPeerCall(
	ctx context.Context,
	call peerCall,
	cause error,
	spent time.Duration,
) {
	if errors.Is(ctx.Err(), context.Canceled) {
		w.observer.PeerCallCancelled(ctx, call.address, call.askedFor, spent)

		return
	}
	w.observer.PeerUnreachable(ctx, call.address, call.askedFor, cause, spent)
}

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
	address                string
	path                   string
	askedFor               peerasks.AskedFor
	amountOfDocumentsAsked int
	form                   url.Values
	headersTimeout         time.Duration
}

type Wire struct {
	client                   *http.Client
	clock                    Clock
	searchedNetwork          SearchedNetwork
	maxResponseBytes         int64
	callsInFlight            peerCallsInFlight
	urlMetadataCallBudget    time.Duration
	searchCallBudget         time.Duration
	searchCallHeadersTimeout time.Duration
	observer                 PeerCallObserver
}

func New(
	client *http.Client,
	clock Clock,
	searchedNetwork SearchedNetwork,
	limits PeerCallLimits,
	observer PeerCallObserver,
) Wire {
	return Wire{
		client:                   client,
		clock:                    clock,
		searchedNetwork:          searchedNetwork,
		maxResponseBytes:         limits.MaxResponseBytes,
		callsInFlight:            newPeerCallsInFlight(limits.PeerCallsInFlight, observer),
		urlMetadataCallBudget:    limits.URLMetadataCallBudget,
		searchCallBudget:         limits.SearchCallBudget,
		searchCallHeadersTimeout: limits.SearchCallHeadersTimeout,
		observer:                 observer,
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
) <-chan peerasks.URLMetadataAskOutcome {
	outcomesAsTheySettle := make(chan peerasks.URLMetadataAskOutcome, len(asks))
	go func() {
		defer close(outcomesAsTheySettle)
		w.callsInFlight.putEveryPeerCallInTheOrderGiven(
			ctx,
			len(asks),
			func(index int) (string, peerasks.AskedFor) {
				return asks[index].Peer.Address, peerasks.URLMetadata
			},
			func(index int) { outcomesAsTheySettle <- w.putURLMetadataAsk(ctx, asks[index]) },
		)
	}()

	return outcomesAsTheySettle
}

func (w Wire) putURLMetadataAsk(
	ctx context.Context,
	ask peerasks.URLMetadataAsk,
) peerasks.URLMetadataAskOutcome {
	ctx, endPeerCall := context.WithTimeout(ctx, w.urlMetadataCallBudget)
	defer endPeerCall()
	startedAt := time.Now()
	outcome := peerasks.URLMetadataAskOutcome{Ask: ask, Put: true}
	response, ok := w.urlMetadataResponse(
		ctx,
		peerCall{
			address:                ask.Peer.Address,
			path:                   yacyproto.PathURLMetadata,
			askedFor:               peerasks.URLMetadata,
			amountOfDocumentsAsked: len(ask.Documents),
			form:                   w.requestForURLMetadata(ask).Form(),
		},
		startedAt,
	)
	if !ok {
		return outcome
	}

	w.observer.PeerAnsweredURLMetadata(
		ctx, ask.Peer.Address, len(ask.Documents), len(response.URLs), time.Since(startedAt),
	)
	outcome.Answer = yacymodel.Some(
		peerasks.AnsweredURLMetadataAsk{Ask: ask, MetadataOfEachDocument: response.URLs},
	)

	return outcome
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
			ctx,
			call.address,
			call.askedFor,
			call.amountOfDocumentsAsked,
			err,
			time.Since(startedAt),
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
	ctx, endPeerCall := context.WithTimeout(ctx, w.searchCallBudget)
	defer endPeerCall()
	startedAt := time.Now()
	response, ok := w.searchResponse(
		ctx,
		peerCall{
			address:                ask.Peer.Address,
			path:                   yacyproto.PathSearch,
			askedFor:               peerasks.SearchDocuments,
			amountOfDocumentsAsked: len(ask.DocumentsToMatch),
			form:                   w.requestForSearchDocuments(ctx, ask).Form(),
			headersTimeout:         w.searchCallHeadersTimeout,
		},
		startedAt,
	)
	if !ok {
		return peerasks.AnsweredSearchDocumentsAsk{}, false
	}

	abstract := response.IndexAbstract[ask.Word]
	matchedDocuments := matchedDocumentsOf(response)
	w.observer.PeerSearchedDocuments(
		ctx,
		ask.Peer.Address,
		len(abstract),
		len(matchedDocuments),
		time.Since(startedAt),
	)

	return peerasks.AnsweredSearchDocumentsAsk{
		Ask:                             ask,
		Abstract:                        abstract,
		MatchedDocuments:                matchedDocuments,
		AmountOfDocumentsHeldForTheWord: amountOfDocumentsHeldForTheWordOf(response, ask.Word),
		PeerSearched:                    response.SearchTime > 0,
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
	if ask.Abstract {
		request.Abstracts = yacyproto.SearchAbstractsOf([]yacymodel.Hash{ask.Word})
	}
	if matchedDocumentsCeiling, asked := ask.MatchedDocumentsCeiling.Get(); asked {
		request.Query = []yacymodel.Hash{ask.Word}
		request.Count = matchedDocumentsCeiling
	}
	request.URLs = ask.DocumentsToMatch

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
			ctx,
			call.address,
			call.askedFor,
			call.amountOfDocumentsAsked,
			err,
			time.Since(startedAt),
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
		w.observer.PeerUnreachable(
			ctx,
			call.address,
			call.askedFor,
			call.amountOfDocumentsAsked,
			err,
			time.Since(startedAt),
		)
		return "", false
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	headersWait := headersWaitWithin(ctx, call.headersTimeout, w.clock)
	defer headersWait.end()
	resp, err := headersWait.responseTo(w.client, req)
	if err != nil {
		w.reportUnansweredPeerCall(ctx, call, headersWait, err, time.Since(startedAt))
		return "", false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		w.observer.PeerRefused(
			ctx,
			call.address,
			call.askedFor,
			call.amountOfDocumentsAsked,
			resp.StatusCode,
			time.Since(startedAt),
		)
		return "", false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, w.maxResponseBytes))
	if err != nil {
		w.reportUnreadAnswer(ctx, call, err, time.Since(startedAt))
		return "", false
	}

	return string(body), true
}

func (w Wire) reportUnansweredPeerCall(
	ctx context.Context,
	call peerCall,
	headersWait *headersWait,
	cause error,
	spent time.Duration,
) {
	if headersWait.headersLate.Load() {
		w.observer.PeerHeadersLate(
			ctx,
			call.address,
			call.askedFor,
			call.amountOfDocumentsAsked,
			spent,
		)

		return
	}
	if callWasCancelled(ctx) {
		w.observer.PeerCallCancelled(
			ctx,
			call.address,
			call.askedFor,
			call.amountOfDocumentsAsked,
			spent,
		)

		return
	}
	w.observer.PeerUnreachable(
		ctx,
		call.address,
		call.askedFor,
		call.amountOfDocumentsAsked,
		cause,
		spent,
	)
}

func callWasCancelled(ctx context.Context) bool {
	return errors.Is(ctx.Err(), context.Canceled)
}

func (w Wire) reportUnreadAnswer(
	ctx context.Context,
	call peerCall,
	cause error,
	spent time.Duration,
) {
	if callWasCancelled(ctx) {
		w.observer.PeerCallCancelled(
			ctx,
			call.address,
			call.askedFor,
			call.amountOfDocumentsAsked,
			spent,
		)

		return
	}
	w.observer.PeerAnswerUnreadable(
		ctx,
		call.address,
		call.askedFor,
		call.amountOfDocumentsAsked,
		cause,
		spent,
	)
}

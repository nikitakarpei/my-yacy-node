// Package peercallwire puts asks to peers over the YaCy search protocol and
// reads back what each peer answered.
package peercallwire

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
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

func (w Wire) AskForMatchedDocuments(
	ctx context.Context,
	asks []peerasks.MatchedDocumentsAsk,
) []peerasks.AnsweredMatchedDocumentsAsk {
	return putAsksToPeers(
		w.callsInFlight,
		asks,
		func(ask peerasks.MatchedDocumentsAsk) (peerasks.AnsweredMatchedDocumentsAsk, bool) {
			return w.putMatchedDocumentsAsk(ctx, ask)
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

func (w Wire) putMatchedDocumentsAsk(
	ctx context.Context,
	ask peerasks.MatchedDocumentsAsk,
) (peerasks.AnsweredMatchedDocumentsAsk, bool) {
	ctx, endPeerCall := context.WithTimeout(ctx, w.peerCallBudget)
	defer endPeerCall()
	startedAt := time.Now()
	response, ok := w.searchResponse(
		ctx,
		peerCall{
			address:  ask.Peer.Address,
			path:     yacyproto.PathSearch,
			askedFor: peerasks.MatchedDocuments,
			form:     w.requestForMatchedDocuments(ctx, ask).Form(),
		},
		startedAt,
	)
	if !ok {
		return peerasks.AnsweredMatchedDocumentsAsk{}, false
	}

	matchedDocuments := matchedDocumentsOf(response)
	w.observer.PeerAnsweredMatchedDocuments(
		ctx, ask.Peer.Address, len(matchedDocuments), time.Since(startedAt),
	)

	return peerasks.AnsweredMatchedDocumentsAsk{Ask: ask, MatchedDocuments: matchedDocuments}, true
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

func wordCountOf(posting yacymodel.Optional[yacymodel.RWIPosting]) queryanswers.WordCount {
	counted, sent := posting.Get()
	if !sent {
		return queryanswers.WordCount{}
	}

	return queryanswers.WordCount{Hits: counted.Hits, TextWords: counted.TextWords}
}

func (w Wire) requestForMatchedDocuments(
	ctx context.Context,
	ask peerasks.MatchedDocumentsAsk,
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

func (w Wire) AskForMatchedAndHeldDocuments(
	ctx context.Context,
	asks []peerasks.MatchedAndHeldDocumentsAsk,
) []peerasks.AnsweredMatchedAndHeldDocumentsAsk {
	return putAsksToPeers(
		w.callsInFlight,
		asks,
		func(ask peerasks.MatchedAndHeldDocumentsAsk) (peerasks.AnsweredMatchedAndHeldDocumentsAsk, bool) {
			return w.putMatchedAndHeldDocumentsAsk(ctx, ask)
		},
	)
}

func (w Wire) putMatchedAndHeldDocumentsAsk(
	ctx context.Context,
	ask peerasks.MatchedAndHeldDocumentsAsk,
) (peerasks.AnsweredMatchedAndHeldDocumentsAsk, bool) {
	ctx, endPeerCall := context.WithTimeout(ctx, w.peerCallBudget)
	defer endPeerCall()
	startedAt := time.Now()
	response, ok := w.searchResponse(
		ctx,
		peerCall{
			address:  ask.Peer.Address,
			path:     yacyproto.PathSearch,
			askedFor: peerasks.MatchedAndHeldDocuments,
			form:     w.requestForMatchedAndHeldDocuments(ctx, ask).Form(),
		},
		startedAt,
	)
	if !ok {
		return peerasks.AnsweredMatchedAndHeldDocumentsAsk{}, false
	}

	documentsListedForTheWord := response.IndexAbstract[ask.Word]
	w.observer.PeerAnsweredMatchedAndHeldDocuments(
		ctx, ask.Peer.Address, len(documentsListedForTheWord), time.Since(startedAt),
	)

	return peerasks.AnsweredMatchedAndHeldDocumentsAsk{
		Ask:                             ask,
		DocumentsListedForTheWord:       documentsListedForTheWord,
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

func (w Wire) requestForMatchedAndHeldDocuments(
	ctx context.Context,
	ask peerasks.MatchedAndHeldDocumentsAsk,
) yacyproto.SearchRequest {
	request := w.requestFor(ctx, ask.ExcludedWords, ask.Language)
	request.Abstracts = yacyproto.SearchAbstractsOf([]yacymodel.Hash{ask.Word})
	request.Query = []yacymodel.Hash{ask.Word}
	request.Count = ask.ItemsCeiling

	return request
}

func (w Wire) AskForCrossCheckedDocuments(
	ctx context.Context,
	asks []peerasks.CrossCheckedDocumentsAsk,
) []peerasks.AnsweredCrossCheckedDocumentsAsk {
	return putAsksToPeers(
		w.callsInFlight,
		asks,
		func(ask peerasks.CrossCheckedDocumentsAsk) (peerasks.AnsweredCrossCheckedDocumentsAsk, bool) {
			return w.putCrossCheckedDocumentsAsk(ctx, ask)
		},
	)
}

func (w Wire) putCrossCheckedDocumentsAsk(
	ctx context.Context,
	ask peerasks.CrossCheckedDocumentsAsk,
) (peerasks.AnsweredCrossCheckedDocumentsAsk, bool) {
	ctx, endPeerCall := context.WithTimeout(ctx, w.peerCallBudget)
	defer endPeerCall()
	startedAt := time.Now()
	response, ok := w.searchResponse(
		ctx,
		peerCall{
			address:  ask.Peer.Address,
			path:     yacyproto.PathSearch,
			askedFor: peerasks.CrossCheckedDocuments,
			form:     w.requestForCrossCheckedDocuments(ctx, ask).Form(),
		},
		startedAt,
	)
	if !ok {
		return peerasks.AnsweredCrossCheckedDocumentsAsk{}, false
	}

	documentsHeldForTheWord := response.IndexAbstract[ask.Word]
	w.observer.PeerAnsweredCrossCheckedDocuments(
		ctx, ask.Peer.Address, len(documentsHeldForTheWord), time.Since(startedAt),
	)

	return peerasks.AnsweredCrossCheckedDocumentsAsk{
		Ask:                     ask,
		DocumentsHeldForTheWord: documentsHeldForTheWord,
	}, true
}

func (w Wire) requestForCrossCheckedDocuments(
	ctx context.Context,
	ask peerasks.CrossCheckedDocumentsAsk,
) yacyproto.SearchRequest {
	request := w.requestFor(ctx, nil, "")
	request.Abstracts = yacyproto.SearchAbstractsOf([]yacymodel.Hash{ask.Word})
	request.URLs = ask.Documents

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

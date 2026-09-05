// Package searchendpoint answers the YaCy search request of a remote peer with
// the documents and match report this node holds.
package searchendpoint

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func Mount(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	results searchresult.Results,
	metrics *searchmetrics.SearchMetrics,
	partitions yacymodel.DHTRingPartitions,
) {
	httpguard.Mount(
		router,
		yacyproto.PathSearch,
		yacyproto.SearchEndpointMethods,
		yacyproto.ParseSearchRequest,
		endpoint{
			identity: identity,
			results:  results,
			observation: searchObservation{
				metrics:      metrics,
				nodePosition: yacymodel.DHTRingPositionOf(identity.Hash),
				partitions:   partitions,
			},
		}.Serve,
	)
}

type endpoint struct {
	identity    nodeidentity.Identity
	results     searchresult.Results
	observation searchObservation
}

func (e endpoint) Serve(
	ctx context.Context,
	req yacyproto.SearchRequest,
) (yacyproto.SearchResponse, error) {
	resp := yacyproto.SearchResponse{}

	if e.identity.NetworkMatches(req.NetworkName) {
		criteria, err := criteriaFromRequest(req)
		if err != nil {
			e.observation.observeInvalidCriteria()

			return yacyproto.SearchResponse{}, fmt.Errorf("search criteria: %w", err)
		}
		requestedIndexAbstracts := requestedIndexAbstractsFromRequest(req)
		if ignoredOptions := ignoredOptionNames(req); len(ignoredOptions) != 0 {
			slog.DebugContext(ctx, "ignoring accepted search options",
				slog.Any("options", ignoredOptions),
			)
			e.observation.observeIgnoredOptions(ignoredOptions)
		}
		searchCtx := ctx
		if criteria.TimeLimit > 0 {
			var cancel func()
			searchCtx, cancel = context.WithTimeout(ctx, criteria.TimeLimit)
			defer cancel()
		}

		result, err := e.results.ResultFor(searchCtx, criteria, requestedIndexAbstracts)
		if err != nil {
			e.observation.observeSearchFailure(err)

			return yacyproto.SearchResponse{}, fmt.Errorf("search: %w", err)
		}
		e.observation.observeServed(result)

		resp.SearchTime = int(result.Duration / time.Millisecond)
		resp.References = strings.Join(result.Topics, ",")
		resp.JoinCount = result.TotalDocumentsMatchingEveryTerm
		resp.Resources = searchResourcesFrom(ctx, result)
		resp.Count = len(resp.Resources)
		resp.IndexCount = result.PostingsHeldPerTerm
		resp.IndexAbstract = encodedIndexAbstractsFrom(result.IndexAbstracts)
	} else {
		e.observation.observeNetworkMismatch()
	}

	slog.DebugContext(ctx, "search completed",
		slog.Int("resultCount", resp.Count),
		slog.Int("joinCount", resp.JoinCount),
	)

	return resp, nil
}

// searchResourcesFrom pairs each document with the posting that matched it. A
// peer drops a result whose posting is missing or names another document, so a
// document this node cannot name by hash is left out of the answer.
func searchResourcesFrom(
	ctx context.Context,
	result searchresult.Result,
) []yacyproto.SearchResource {
	resources := make([]yacyproto.SearchResource, 0, len(result.DocumentMetadata))
	for _, metadata := range result.DocumentMetadata {
		documentHash, err := metadata.Hash()
		if err != nil {
			slog.WarnContext(ctx, "search result withheld",
				slog.String("reason", "document hash unknown"),
				slog.Any("error", err),
			)

			continue
		}
		resources = append(resources, yacyproto.SearchResource{
			Metadata: metadata,
			Posting:  result.PostingPerDocument[documentHash],
		})
	}

	return resources
}

func encodedIndexAbstractsFrom(
	documentsPerTerm map[yacymodel.Hash][]yacymodel.URLHash,
) map[yacymodel.Hash]string {
	abstracts := make(map[yacymodel.Hash]string, len(documentsPerTerm))
	for term, documentHashes := range documentsPerTerm {
		abstracts[term] = yacyproto.EncodeSearchIndexAbstract(documentHashes)
	}

	return abstracts
}

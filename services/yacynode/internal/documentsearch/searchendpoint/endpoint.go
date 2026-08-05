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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func Mount(
	router httpguard.WireRouter,
	identity nodeidentity.Identity,
	results searchresult.Results,
) {
	httpguard.Mount(
		router,
		yacyproto.PathSearch,
		yacyproto.SearchEndpointMethods,
		yacyproto.ParseSearchRequest,
		endpoint{identity: identity, results: results}.Serve,
	)
}

type endpoint struct {
	identity nodeidentity.Identity
	results  searchresult.Results
}

func (e endpoint) Serve(
	ctx context.Context,
	req yacyproto.SearchRequest,
) (yacyproto.SearchResponse, error) {
	resp := yacyproto.SearchResponse{}

	if e.identity.NetworkMatches(req.NetworkName) {
		criteria, err := criteriaFromRequest(req)
		if err != nil {
			return yacyproto.SearchResponse{}, fmt.Errorf("search criteria: %w", err)
		}
		requestedReport := requestedReportFromRequest(req)
		if ignoredOptions := ignoredOptionNames(req); len(ignoredOptions) != 0 {
			slog.DebugContext(ctx, "ignoring accepted search options",
				slog.Any("options", ignoredOptions),
			)
		}
		searchCtx := ctx
		if criteria.TimeLimit > 0 {
			var cancel func()
			searchCtx, cancel = context.WithTimeout(ctx, criteria.TimeLimit)
			defer cancel()
		}

		result, err := e.results.ResultFor(searchCtx, criteria, requestedReport)
		if err != nil {
			return yacyproto.SearchResponse{}, fmt.Errorf("search: %w", err)
		}

		resp.SearchTime = int(result.Duration / time.Millisecond)
		resp.References = strings.Join(result.Topics, ",")
		resp.JoinCount = result.TotalDocumentsMatchingEveryTerm
		resp.Count = len(result.DocumentMetadata)
		resp.Resources = result.DocumentMetadata
		resp.IndexCount = result.TotalMatchesPerTerm
		resp.IndexAbstract = indexAbstractFrom(result.DocumentsMatchingEachReportedTerm)
	}

	slog.DebugContext(ctx, "search completed",
		slog.Int("resultCount", resp.Count),
		slog.Int("joinCount", resp.JoinCount),
	)

	return resp, nil
}

func indexAbstractFrom(
	documentsMatchingEachReportedTerm map[yacymodel.Hash][]yacymodel.URLHash,
) map[yacymodel.Hash]string {
	abstracts := make(map[yacymodel.Hash]string, len(documentsMatchingEachReportedTerm))
	for term, documentHashes := range documentsMatchingEachReportedTerm {
		abstracts[term] = yacyproto.EncodeSearchIndexAbstract(documentHashes)
	}

	return abstracts
}

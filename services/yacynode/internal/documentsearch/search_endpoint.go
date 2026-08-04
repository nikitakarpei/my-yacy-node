package documentsearch

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	defaultSearchCount = 10
	defaultSearchTime  = 3 * time.Second
)

type searchEndpoint struct {
	identity nodeidentity.Identity
	searcher searcher
}

func (e searchEndpoint) Serve(
	ctx context.Context,
	req yacyproto.SearchRequest,
) (yacyproto.SearchResponse, error) {
	resp := yacyproto.SearchResponse{}

	if e.identity.NetworkMatches(req.NetworkName) {
		criteria, err := searchCriteriaFromRequest(req)
		if err != nil {
			return yacyproto.SearchResponse{}, fmt.Errorf("search criteria: %w", err)
		}
		if ignoredOptions := ignoredOptionNames(req); len(ignoredOptions) != 0 {
			slog.DebugContext(ctx, "ignoring accepted search options",
				slog.Any("options", ignoredOptions),
			)
		}
		searchCtx := ctx
		if criteria.timeLimit > 0 {
			var cancel func()
			searchCtx, cancel = context.WithTimeout(ctx, criteria.timeLimit)
			defer cancel()
		}

		result, err := e.searcher.search(searchCtx, criteria)
		if err != nil {
			return yacyproto.SearchResponse{}, fmt.Errorf("search: %w", err)
		}

		resp.SearchTime = int(result.searchDuration / time.Millisecond)
		resp.References = strings.Join(result.topics, ",")
		resp.JoinCount = result.totalDocumentsMatchingEveryTerm
		resp.Count = len(result.documentMetadata)
		resp.Resources = result.documentMetadata
		resp.IndexCount = result.totalMatchesPerTerm
		resp.IndexAbstract = indexAbstractFrom(result.documentsMatchingEachReportedTerm)
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

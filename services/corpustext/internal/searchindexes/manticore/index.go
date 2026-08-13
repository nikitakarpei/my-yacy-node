package manticore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/crawledpagedocument"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type Index struct {
	endpoint string
	tables   languageindex.LanguageIndexes
	client   *http.Client
}

func New(
	endpoint string,
	tables languageindex.LanguageIndexes,
	client *http.Client,
) *Index {
	return &Index{
		endpoint: strings.TrimRight(endpoint, "/"),
		tables:   tables,
		client:   client,
	}
}

type replaceRequest struct {
	Table    string                  `json:"table"`
	Identity int64                   `json:"id"`
	Document searchdocument.Document `json:"doc"`
}

func (idx *Index) Index(
	ctx context.Context,
	page yacycrawlcontract.PageTextRepresentation,
) error {
	identity := documentIdentity(page.CanonicalURL)
	body, err := json.Marshal(replaceRequest{
		Table:    idx.tables.NameFor(page.Language),
		Identity: identity,
		Document: crawledpagedocument.Of(page),
	})
	if err != nil {
		return fmt.Errorf("marshal search document %d: %w", identity, err)
	}
	target := idx.endpoint + "/replace"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build index request %d: %w", identity, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := idx.client.Do(req)
	if err != nil {
		return fmt.Errorf("index document %d: %w", identity, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf(
			"index document %d: status %d: %s",
			identity,
			resp.StatusCode,
			detail,
		)
	}
	return nil
}

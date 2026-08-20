package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/crawledpagedocument"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type Index struct {
	endpoint string
	indexes  languageindex.LanguageIndexes
	client   *http.Client
}

func New(
	endpoint string,
	indexes languageindex.LanguageIndexes,
	client *http.Client,
) *Index {
	return &Index{
		endpoint: strings.TrimRight(endpoint, "/"),
		indexes:  indexes,
		client:   client,
	}
}

func (idx *Index) Index(
	ctx context.Context,
	page yacycrawlcontract.PageTextRepresentation,
) error {
	identity := documentIdentity(page.CanonicalURL.String())
	body, err := json.Marshal(crawledpagedocument.Of(page))
	if err != nil {
		return fmt.Errorf("marshal search document %s: %w", identity, err)
	}
	target := fmt.Sprintf(
		"%s/%s/_doc/%s",
		idx.endpoint,
		idx.indexes.NameFor(page.Language),
		url.PathEscape(identity),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, target, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build index request %s: %w", identity, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := idx.client.Do(req)
	if err != nil {
		return fmt.Errorf("index document %s: %w", identity, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf(
			"index document %s: status %d: %s",
			identity,
			resp.StatusCode,
			detail,
		)
	}
	return nil
}

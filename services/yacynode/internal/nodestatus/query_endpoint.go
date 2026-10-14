package nodestatus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	msgCountFailed = "count failed"
	msgCountServed = "count served"
)

var (
	errWordAmount    = errors.New("estimate words of stored postings")
	errPostingAmount = errors.New("count stored postings of the asked word")
	errLURLCount     = errors.New("count stored URL metadata records")
)

type queryEndpoint struct {
	identity       nodeidentity.Identity
	now            func() time.Time
	vault          *vault.Vault
	words          WordAmountEstimate
	postingAmounts PostingAmountCounter
	urls           URLCounter
}

func (e queryEndpoint) Serve(
	ctx context.Context,
	req yacyproto.QueryRequest,
) (yacyproto.QueryResponse, error) {
	resp := yacyproto.QueryResponse{
		Response: yacyproto.QueryResponseRejected,
		MyTime:   e.now(),
		Magic:    magicOf(e.identity),
	}

	if e.identity.Addresses(req.NetworkName, req.YouAre) {
		count, err := e.count(ctx, req)
		if err != nil {
			return yacyproto.QueryResponse{}, fmt.Errorf("%s: %w", msgCountFailed, err)
		}
		resp.Response = count
	}

	slog.DebugContext(ctx, msgCountServed,
		slog.String("object", string(req.Object)),
		slog.String("env", req.Env),
		slog.Int("count", resp.Response),
	)

	return resp, nil
}

func magicOf(identity nodeidentity.Identity) string {
	return strconv.FormatInt(identity.Start.UnixMilli(), 10)
}

func (e queryEndpoint) count(ctx context.Context, req yacyproto.QueryRequest) (int, error) {
	var count int
	err := e.vault.View(ctx, func(tx *vault.Txn) error {
		measured, err := e.countOf(tx, req)
		count = measured

		return err
	})

	return count, err
}

func (e queryEndpoint) countOf(tx *vault.Txn, req yacyproto.QueryRequest) (int, error) {
	switch req.Object {
	case yacyproto.ObjectRWICount:
		return e.amountOfWords(tx)
	case yacyproto.ObjectRWIURLCount:
		return e.postingAmountOf(tx, req.Env)
	case yacyproto.ObjectLURLCount:
		return e.urlCount(tx)
	default:
		return 0, nil
	}
}

func (e queryEndpoint) amountOfWords(tx *vault.Txn) (int, error) {
	n, err := e.words.EstimatedAmountOfWords(tx)

	return n, wrapCount(errWordAmount, err)
}

func (e queryEndpoint) postingAmountOf(tx *vault.Txn, env string) (int, error) {
	word, err := yacymodel.ParseHash(env)
	if err != nil {
		return 0, nil //nolint:nilerr // an env that names no word asks for no postings
	}
	n, err := e.postingAmounts.AmountOfPostingsOf(tx, word)

	return n, wrapCount(errPostingAmount, err)
}

func (e queryEndpoint) urlCount(tx *vault.Txn) (int, error) {
	n, err := e.urls.Count(tx)

	return n, wrapCount(errLURLCount, err)
}

func wrapCount(sentinel error, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%w: %w", sentinel, err)
}

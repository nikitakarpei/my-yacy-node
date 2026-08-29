// Package grpc reads one page's markdown from the corpus that holds it, over the corpus
// contract. A page the corpus does not hold is an answer and not a failure, so this package
// turns the contract's NOT_FOUND into that answer.
package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	corpusmarkdownv1 "github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/corpusmarkdown/v1"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
)

type MarkdownCorpus struct {
	connection     *grpc.ClientConn
	corpus         corpusmarkdownv1.MarkdownCorpusClient
	recallDeadline time.Duration
}

func OpenMarkdownCorpus(
	corpusAddress string,
	recallDeadline time.Duration,
) (*MarkdownCorpus, error) {
	connection, err := grpc.NewClient(
		corpusAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("open the markdown corpus at %s: %w", corpusAddress, err)
	}
	return &MarkdownCorpus{
		connection:     connection,
		corpus:         corpusmarkdownv1.NewMarkdownCorpusClient(connection),
		recallDeadline: recallDeadline,
	}, nil
}

func (c *MarkdownCorpus) PageMarkdownAt(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) (pageread.PageMarkdown, error) {
	recallCtx, stopRecalling := context.WithTimeout(ctx, c.recallDeadline)
	defer stopRecalling()

	recalled, err := c.corpus.RecallPage(recallCtx, &corpusmarkdownv1.RecallPageRequest{
		Url: pageURL.String(),
	})
	if status.Code(err) == codes.NotFound {
		return pageread.PageMarkdown{}, pageread.ErrPageNotInCorpus
	}
	if err != nil {
		return pageread.PageMarkdown{}, fmt.Errorf("recall %q: %w", pageURL, err)
	}
	return pageMarkdownFrom(recalled), nil
}

func pageMarkdownFrom(
	recalled *corpusmarkdownv1.RecallPageResponse,
) pageread.PageMarkdown {
	return pageread.PageMarkdown{
		Markdown: recalled.GetMarkdown(),
		Version:  recalled.GetVersion(),
		StoredAt: recalled.GetStoredAt().AsTime(),
	}
}

func (c *MarkdownCorpus) Close() error {
	if err := c.connection.Close(); err != nil {
		return fmt.Errorf("close the markdown corpus connection: %w", err)
	}
	return nil
}

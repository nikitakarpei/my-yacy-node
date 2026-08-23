// Package markdowncorpusclienttest dials the markdown corpus contract of a running service in a test.
package markdowncorpusclienttest

import (
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	corpusmarkdownv1 "github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/corpusmarkdown/v1"
)

func New(t *testing.T, listenAddress string) corpusmarkdownv1.MarkdownCorpusClient {
	t.Helper()
	connection, err := grpc.NewClient(
		listenAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial markdown corpus: %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	return corpusmarkdownv1.NewMarkdownCorpusClient(connection)
}

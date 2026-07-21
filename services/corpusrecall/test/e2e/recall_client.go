//go:build e2e

package e2e

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

func dialRecall(t *testing.T, addr string) corpusrecallv1.RecallClient {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial corpusrecall: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return corpusrecallv1.NewRecallClient(conn)
}

func recall(
	t *testing.T,
	ctx context.Context,
	client corpusrecallv1.RecallClient,
	url string,
) *corpusrecallv1.RecallResponse {
	t.Helper()
	resp, err := client.Recall(ctx, &corpusrecallv1.RecallRequest{
		Url: url,
		Kinds: []corpusrecallv1.RepresentationKind{
			corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN,
		},
	})
	if err != nil {
		t.Fatalf("recall %q: %v", url, err)
	}
	return resp
}

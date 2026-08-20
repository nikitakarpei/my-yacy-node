// Package recallclienttest dials the recall contract of a running service in a test.
package recallclienttest

import (
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

func New(t *testing.T, listenAddress string) corpusrecallv1.RecallClient {
	t.Helper()
	connection, err := grpc.NewClient(
		listenAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial recall: %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	return corpusrecallv1.NewRecallClient(connection)
}

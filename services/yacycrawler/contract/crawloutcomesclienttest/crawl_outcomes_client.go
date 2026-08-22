// Package crawloutcomesclienttest dials the crawl outcomes contract of a running service in a test.
package crawloutcomesclienttest

import (
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	crawlerv1 "github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/crawler/v1"
)

func New(t *testing.T, listenAddress string) crawlerv1.CrawlOutcomesClient {
	t.Helper()
	connection, err := grpc.NewClient(
		listenAddress, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial crawl outcomes: %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	return crawlerv1.NewCrawlOutcomesClient(connection)
}

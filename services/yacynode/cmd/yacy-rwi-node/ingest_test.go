package main_test

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	yacynode "github.com/nikitakarpei/yacy-rwi-node/yacynode/cmd/yacy-rwi-node"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const ingestStreamCapacity = 64

func TestCrawledURLMetadataReachesTheNodesURLDirectory(t *testing.T) {
	broker := natstestserver.Start(t)
	createIngestStream(t, broker)

	config := nodeConfigFor(t)
	config.Crawl = crawlConfigFor(broker)

	node := startNode(t, config)
	defer node.stop()

	publishIngestChunk(t, broker, yacycrawlcontract.PageRWIMetadataChunk{
		CanonicalURL: "https://example.org",
		Metadata:     []yacymodel.URLMetadata{{Address: "https://example.org"}},
	})

	awaitURLCount(t, node, 1)
}

func TestRunNodeReportsAnUnreachableCrawlBroker(t *testing.T) {
	config := nodeConfigFor(t)
	config.Crawl = crawlConfigFor("nats://127.0.0.1:1")

	node := startNode(t, config)
	defer node.stop()

	if err := node.wait(t); err == nil {
		t.Fatal("RunNode returned nil, want the broker failure")
	}
}

func createIngestStream(t *testing.T, broker string) {
	t.Helper()

	js := natstestserver.ConnectJetStream(t, broker)
	if _, err := js.CreateOrUpdateStream(t.Context(), jetstream.StreamConfig{
		Name: yacycrawlcontract.CrawledPageStreamName(
			yacycrawlcontract.PageRepresentationKindRWI,
		),
		Subjects:  []string{yacynode.DefaultIngestSubject},
		Retention: jetstream.WorkQueuePolicy,
		MaxMsgs:   ingestStreamCapacity,
		Discard:   jetstream.DiscardNew,
	}); err != nil {
		t.Fatalf("create ingest stream: %v", err)
	}
}

func crawlConfigFor(broker string) yacynode.CrawlConfig {
	return yacynode.CrawlConfig{
		NATSURL:       broker,
		IngestSubject: yacynode.DefaultIngestSubject,
		IngestDurable: yacynode.DefaultIngestDurable,
	}
}

func publishIngestChunk(
	t *testing.T,
	broker string,
	chunk yacycrawlcontract.PageRWIMetadataChunk,
) {
	t.Helper()

	data, err := yacycrawlcontract.MarshalPageRWIChunk(chunk)
	if err != nil {
		t.Fatalf("marshal ingest chunk: %v", err)
	}
	js := natstestserver.ConnectJetStream(t, broker)
	if _, err := js.Publish(t.Context(), yacynode.DefaultIngestSubject, data); err != nil {
		t.Fatalf("publish ingest chunk: %v", err)
	}
}

func awaitURLCount(t *testing.T, node runningNode, want int) {
	t.Helper()

	deadline := time.Now().Add(settleFor)
	var counted int
	for time.Now().Before(deadline) {
		counted = node.query(t, yacyproto.ObjectLURLCount).Response
		if counted == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("url count = %d, want %d", counted, want)
}

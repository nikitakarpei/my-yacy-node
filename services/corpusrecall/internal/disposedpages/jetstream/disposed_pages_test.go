package jetstream_test

import (
	"context"
	"testing"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	disposedpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/disposedpages/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	canonicalURL      = "https://example.com/"
	otherCanonicalURL = "https://example.org/"
)

func emptyDisposedPagesBucket(t *testing.T) natsjetstream.KeyValue {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	if _, err := js.CreateKeyValue(context.Background(), natsjetstream.KeyValueConfig{
		Bucket: yacycrawlcontract.DisposedPagesBucketName,
	}); err != nil {
		t.Fatalf("create disposed pages bucket: %v", err)
	}
	bucket, err := js.KeyValue(context.Background(), yacycrawlcontract.DisposedPagesBucketName)
	if err != nil {
		t.Fatalf("open disposed pages bucket: %v", err)
	}
	return bucket
}

func disposeOfPage(t *testing.T, bucket natsjetstream.KeyValue, pageURL string) {
	t.Helper()
	if _, err := bucket.Put(
		context.Background(), yacycrawlcontract.DisposedPageKey(pageURL), nil,
	); err != nil {
		t.Fatalf("record disposal: %v", err)
	}
}

func disposalOfPage(t *testing.T, bucket natsjetstream.KeyValue) recall.PageDisposal {
	t.Helper()
	disposal, err := disposedpagesjetstream.NewDisposedPages(bucket).DisposalOf(
		context.Background(), canonicalURL,
	)
	if err != nil {
		t.Fatalf("disposal of %q: %v", canonicalURL, err)
	}
	return disposal
}

func hasDisposalOccurred(t *testing.T, disposal recall.PageDisposal) bool {
	t.Helper()
	occurred, err := disposal.HasOccurred(context.Background())
	if err != nil {
		t.Fatalf("has disposal occurred: %v", err)
	}
	return occurred
}

func TestPageIsNotDisposedWhileTheCrawlerRecordsNothing(t *testing.T) {
	disposal := disposalOfPage(t, emptyDisposedPagesBucket(t))

	if hasDisposalOccurred(t, disposal) {
		t.Error("page reported disposed with no disposal recorded")
	}
}

func TestPageIsNotDisposedWhenTheDisposalPredatesTheLookup(t *testing.T) {
	bucket := emptyDisposedPagesBucket(t)
	disposeOfPage(t, bucket, canonicalURL)

	disposal := disposalOfPage(t, bucket)

	if hasDisposalOccurred(t, disposal) {
		t.Error("page reported disposed by a disposal older than the lookup")
	}
}

func TestPageIsDisposedWhenTheCrawlerDisposesOfItAfterTheLookup(t *testing.T) {
	bucket := emptyDisposedPagesBucket(t)
	disposeOfPage(t, bucket, canonicalURL)
	disposal := disposalOfPage(t, bucket)

	disposeOfPage(t, bucket, canonicalURL)

	if !hasDisposalOccurred(t, disposal) {
		t.Error("page reported kept after the crawler disposed of it")
	}
}

func TestPageIsNotDisposedWhenTheCrawlerDisposesOfAnotherPage(t *testing.T) {
	bucket := emptyDisposedPagesBucket(t)
	disposal := disposalOfPage(t, bucket)

	disposeOfPage(t, bucket, otherCanonicalURL)

	if hasDisposalOccurred(t, disposal) {
		t.Error("page reported disposed by the disposal of another page")
	}
}

func TestTheDisposalOfAPageIsUnknownWhenTheBucketCannotBeRead(t *testing.T) {
	pages := disposedpagesjetstream.NewDisposedPages(emptyDisposedPagesBucket(t))
	abandoned, abandon := context.WithCancel(context.Background())
	abandon()

	if _, err := pages.DisposalOf(abandoned, canonicalURL); err == nil {
		t.Fatal("expected an error when the bucket cannot be read")
	}
}

func TestWhetherADisposalHasOccurredIsUnknownWhenTheBucketCannotBeRead(t *testing.T) {
	disposal := disposalOfPage(t, emptyDisposedPagesBucket(t))
	abandoned, abandon := context.WithCancel(context.Background())
	abandon()

	if _, err := disposal.HasOccurred(abandoned); err == nil {
		t.Fatal("expected an error when the bucket cannot be read")
	}
}

package jetstream_test

import (
	"context"
	"testing"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
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
		context.Background(),
		yacycrawlcontract.DisposedPageKey(canonicalurltest.CanonicalURLOf(t, pageURL)),
		nil,
	); err != nil {
		t.Fatalf("record disposal: %v", err)
	}
}

func disposalMarkOfPage(t *testing.T, bucket natsjetstream.KeyValue) recall.DisposalMark {
	t.Helper()
	mark, err := disposedpagesjetstream.NewDisposedPages(bucket).DisposalMarkOf(
		context.Background(), canonicalurltest.CanonicalURLOf(t, canonicalURL),
	)
	if err != nil {
		t.Fatalf("disposal mark of %q: %v", canonicalURL, err)
	}
	return mark
}

func disposalOccurredSince(
	t *testing.T,
	bucket natsjetstream.KeyValue,
	mark recall.DisposalMark,
) bool {
	t.Helper()
	occurred, err := disposedpagesjetstream.NewDisposedPages(bucket).DisposalOccurredSince(
		context.Background(), canonicalurltest.CanonicalURLOf(t, canonicalURL), mark,
	)
	if err != nil {
		t.Fatalf("disposal occurred since %q: %v", mark, err)
	}
	return occurred
}

func TestPageIsNotDisposedWhileTheCrawlerRecordsNothing(t *testing.T) {
	bucket := emptyDisposedPagesBucket(t)

	if disposalOccurredSince(t, bucket, disposalMarkOfPage(t, bucket)) {
		t.Error("page reported disposed with no disposal recorded")
	}
}

func TestPageIsNotDisposedWhenTheDisposalPredatesTheMark(t *testing.T) {
	bucket := emptyDisposedPagesBucket(t)
	disposeOfPage(t, bucket, canonicalURL)

	mark := disposalMarkOfPage(t, bucket)

	if disposalOccurredSince(t, bucket, mark) {
		t.Error("page reported disposed by a disposal older than the mark")
	}
}

func TestPageIsDisposedWhenTheCrawlerDisposesOfItAfterTheMark(t *testing.T) {
	bucket := emptyDisposedPagesBucket(t)
	disposeOfPage(t, bucket, canonicalURL)
	mark := disposalMarkOfPage(t, bucket)

	disposeOfPage(t, bucket, canonicalURL)

	if !disposalOccurredSince(t, bucket, mark) {
		t.Error("page reported kept after the crawler disposed of it")
	}
}

func TestPageIsNotDisposedWhenTheCrawlerDisposesOfAnotherPage(t *testing.T) {
	bucket := emptyDisposedPagesBucket(t)
	mark := disposalMarkOfPage(t, bucket)

	disposeOfPage(t, bucket, otherCanonicalURL)

	if disposalOccurredSince(t, bucket, mark) {
		t.Error("page reported disposed by the disposal of another page")
	}
}

func TestTheDisposalMarkOfAPageIsUnknownWhenTheBucketCannotBeRead(t *testing.T) {
	pages := disposedpagesjetstream.NewDisposedPages(emptyDisposedPagesBucket(t))
	abandoned, abandon := context.WithCancel(context.Background())
	abandon()

	if _, err := pages.DisposalMarkOf(
		abandoned,
		canonicalurltest.CanonicalURLOf(t, canonicalURL),
	); err == nil {
		t.Fatal("expected an error when the bucket cannot be read")
	}
}

func TestWhetherADisposalOccurredIsUnknownWhenTheBucketCannotBeRead(t *testing.T) {
	pages := disposedpagesjetstream.NewDisposedPages(emptyDisposedPagesBucket(t))
	abandoned, abandon := context.WithCancel(context.Background())
	abandon()

	if _, err := pages.DisposalOccurredSince(
		abandoned, canonicalurltest.CanonicalURLOf(t, canonicalURL), "",
	); err == nil {
		t.Fatal("expected an error when the bucket cannot be read")
	}
}

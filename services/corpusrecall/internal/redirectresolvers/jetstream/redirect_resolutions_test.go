package jetstream_test

import (
	"context"
	"testing"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	redirectresolversjetstream "github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/redirectresolvers/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/canonicalurltest"
)

const canonicalURL = "https://example.com/"

func recordedRedirects(t *testing.T) natsjetstream.KeyValue {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	if _, err := js.CreateKeyValue(context.Background(), natsjetstream.KeyValueConfig{
		Bucket: yacycrawlcontract.RedirectResolutionBucketName,
	}); err != nil {
		t.Fatalf("create redirect resolution bucket: %v", err)
	}
	bucket, err := js.KeyValue(
		context.Background(), yacycrawlcontract.RedirectResolutionBucketName,
	)
	if err != nil {
		t.Fatalf("open redirect resolution bucket: %v", err)
	}
	return bucket
}

func TestResolvedURLIsTheCanonicalURLWhenNoRedirectIsRecorded(t *testing.T) {
	resolutions := redirectresolversjetstream.NewRedirectResolutions(recordedRedirects(t))

	got, err := resolutions.ResolvedURLOf(
		context.Background(), canonicalurltest.CanonicalURLOf(t, canonicalURL),
	)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if got.String() != canonicalURL {
		t.Errorf("resolved = %q, want %q", got, canonicalURL)
	}
}

func TestResolvedURLIsTheRedirectTargetTheCrawlerRecorded(t *testing.T) {
	const target = "https://example.com/final"
	bucket := recordedRedirects(t)
	if _, err := bucket.Put(
		context.Background(),
		yacycrawlcontract.RedirectResolutionKey(canonicalurltest.CanonicalURLOf(t, canonicalURL)),
		[]byte(target),
	); err != nil {
		t.Fatalf("record redirect: %v", err)
	}
	resolutions := redirectresolversjetstream.NewRedirectResolutions(bucket)

	got, err := resolutions.ResolvedURLOf(
		context.Background(), canonicalurltest.CanonicalURLOf(t, canonicalURL),
	)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if got.String() != target {
		t.Errorf("resolved = %q, want %q", got, target)
	}
}

func TestResolvedURLIsUnknownWhenTheBucketCannotBeRead(t *testing.T) {
	resolutions := redirectresolversjetstream.NewRedirectResolutions(recordedRedirects(t))
	abandoned, abandon := context.WithCancel(context.Background())
	abandon()

	if _, err := resolutions.ResolvedURLOf(
		abandoned,
		canonicalurltest.CanonicalURLOf(t, canonicalURL),
	); err == nil {
		t.Fatal("expected an error when the bucket cannot be read")
	}
}

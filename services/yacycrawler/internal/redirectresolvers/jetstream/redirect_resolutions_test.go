package jetstream_test

import (
	"context"
	"testing"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/redirectresolvers/jetstream"
)

func redirectResolutionsUnderTest(t *testing.T) *jetstream.RedirectResolutions {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	if _, err := js.CreateKeyValue(context.Background(), natsjetstream.KeyValueConfig{
		Bucket: yacycrawlcontract.RedirectResolutionBucketName,
	}); err != nil {
		t.Fatal(err)
	}
	bucket, err := js.KeyValue(context.Background(), yacycrawlcontract.RedirectResolutionBucketName)
	if err != nil {
		t.Fatal(err)
	}
	return jetstream.NewRedirectResolutions(bucket)
}

func TestResolvedURLOfYieldsTheURLTheRequestedOneRedirectedTo(t *testing.T) {
	resolutions := redirectResolutionsUnderTest(t)
	requested := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")
	canonical := canonicalurltest.CanonicalURLOf(t, "http://example.com/c")
	if err := resolutions.Record(context.Background(), requested, canonical); err != nil {
		t.Fatalf("record: %v", err)
	}

	resolvedURL, err := resolutions.ResolvedURLOf(context.Background(), requested)
	if err != nil {
		t.Fatalf("resolved url of: %v", err)
	}

	if resolvedURL != canonical {
		t.Fatalf("resolvedURL = %q, want %q", resolvedURL, canonical)
	}
}

func TestResolvedURLOfYieldsTheURLItselfWhenNoRedirectWasRecorded(t *testing.T) {
	resolutions := redirectResolutionsUnderTest(t)
	url := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")

	resolvedURL, err := resolutions.ResolvedURLOf(context.Background(), url)
	if err != nil {
		t.Fatalf("resolved url of: %v", err)
	}

	if resolvedURL != url {
		t.Fatalf("resolvedURL = %q, want %q", resolvedURL, url)
	}
}

func TestRecordKeepsAURLThatResolvesToItselfUnrecorded(t *testing.T) {
	resolutions := redirectResolutionsUnderTest(t)
	url := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")

	if err := resolutions.Record(context.Background(), url, url); err != nil {
		t.Fatalf("record: %v", err)
	}

	resolvedURL, err := resolutions.ResolvedURLOf(context.Background(), url)
	if err != nil {
		t.Fatalf("resolved url of: %v", err)
	}
	if resolvedURL != url {
		t.Fatalf("resolvedURL = %q, want %q", resolvedURL, url)
	}
}

package pagemarkdownstore_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

func TestObjectNameIsDeterministicPerURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{name: "root", url: "https://example.test/"},
		{name: "path", url: "https://example.test/page"},
		{name: "query", url: "https://example.test/page?a=1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first := pagemarkdownstore.ObjectName(tc.url)
			second := pagemarkdownstore.ObjectName(tc.url)
			if first != second {
				t.Fatalf("ObjectName not deterministic: %q vs %q", first, second)
			}
			if len(first) != 64 {
				t.Fatalf("ObjectName length = %d, want 64", len(first))
			}
		})
	}
}

func TestObjectNameDistinguishesURLs(t *testing.T) {
	if pagemarkdownstore.ObjectName("https://example.test/a") ==
		pagemarkdownstore.ObjectName("https://example.test/b") {
		t.Fatal("distinct URLs collided to one object name")
	}
}

func TestEnsureBucketStoresAndOverwritesLatest(t *testing.T) {
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	ctx := context.Background()

	store, err := pagemarkdownstore.EnsureBucket(ctx, js)
	if err != nil {
		t.Fatalf("EnsureBucket: %v", err)
	}

	name := pagemarkdownstore.ObjectName("https://example.test/page")
	if _, err := store.PutBytes(ctx, name, []byte("# first")); err != nil {
		t.Fatalf("put first: %v", err)
	}
	got, err := store.GetBytes(ctx, name)
	if err != nil {
		t.Fatalf("get first: %v", err)
	}
	if string(got) != "# first" {
		t.Fatalf("stored markdown = %q, want %q", got, "# first")
	}

	if _, err := store.PutBytes(ctx, name, []byte("# second")); err != nil {
		t.Fatalf("put second: %v", err)
	}
	got, err = store.GetBytes(ctx, name)
	if err != nil {
		t.Fatalf("get second: %v", err)
	}
	if string(got) != "# second" {
		t.Fatalf("latest markdown = %q, want %q", got, "# second")
	}
}

func TestEnsureBucketIsIdempotent(t *testing.T) {
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	ctx := context.Background()
	if _, err := pagemarkdownstore.EnsureBucket(ctx, js); err != nil {
		t.Fatalf("first EnsureBucket: %v", err)
	}
	if _, err := pagemarkdownstore.EnsureBucket(ctx, js); err != nil {
		t.Fatalf("second EnsureBucket: %v", err)
	}
}

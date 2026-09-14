package urlmeta_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func servedURLMetadata(
	t *testing.T,
	servedURLMetadataPerRequest int,
	stored []yacymodel.URLMetadata,
	req yacyproto.URLMetadataRequest,
) []yacymodel.URLMetadata {
	t.Helper()

	v, module := openModule(t, 0)
	if _, err := module.Receiver.Receive(context.Background(), stored); err != nil {
		t.Fatalf("Receive: %v", err)
	}

	mux := http.NewServeMux()
	router := httpguard.NewWireRouter(mux, httpguard.WireGate{
		Guard: httpguard.NewRequestGuard(
			httpguard.DefaultMaxBodyBytes,
			httpguard.DefaultRequestTimeout,
		),
		Respond: httpguard.NewWireResponder(stubRuntimeStatus{}),
		Address: httpguard.NewClientAddressResolver(nil),
	})
	urlmeta.MountURLMetadataLookup(
		router, localIdentity(), v, module.Directory, servedURLMetadataPerRequest,
	)

	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		yacyproto.PathURLMetadata,
		strings.NewReader(req.Form().Encode()),
	)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %q", rec.Code, rec.Body.String())
	}

	resp, err := yacyproto.ParseURLMetadataResponse(context.Background(), rec.Body.Bytes())
	if err != nil {
		t.Fatalf("ParseURLMetadataResponse: %v", err)
	}

	return resp.URLs
}

func addressesOf(metadata []yacymodel.URLMetadata) []string {
	addresses := make([]string, 0, len(metadata))
	for _, url := range metadata {
		addresses = append(addresses, url.Address)
	}

	return addresses
}

func TestURLMetadataLookupServesTheStoredURLsInTheAskedOrder(t *testing.T) {
	first, second := urlMetadata(t, "a"), urlMetadata(t, "b")
	first.Title = "Rock & Roll"

	served := servedURLMetadata(t, 10, []yacymodel.URLMetadata{first, second},
		yacyproto.URLMetadataRequest{
			NetworkName: "freeworld",
			URLs: []yacymodel.URLHash{
				second.Hash,
				urlHashOf(t, "http://absent/"),
				first.Hash,
			},
		},
	)

	if got := addressesOf(served); strings.Join(got, " ") != second.Address+" "+first.Address {
		t.Fatalf("served = %v, want the stored urls in the asked order", got)
	}
	if served[1].Title != first.Title {
		t.Fatalf("title = %q, want %q", served[1].Title, first.Title)
	}
}

func TestURLMetadataLookupServesNothingToAnotherNetwork(t *testing.T) {
	stored := urlMetadata(t, "a")

	served := servedURLMetadata(t, 10, []yacymodel.URLMetadata{stored},
		yacyproto.URLMetadataRequest{
			NetworkName: "othernetwork",
			URLs:        []yacymodel.URLHash{stored.Hash},
		},
	)

	if len(served) != 0 {
		t.Fatalf("served = %v, want nothing", addressesOf(served))
	}
}

func TestURLMetadataLookupServesAtMostTheURLsPerRequest(t *testing.T) {
	first, second := urlMetadata(t, "a"), urlMetadata(t, "b")

	served := servedURLMetadata(t, 1, []yacymodel.URLMetadata{first, second},
		yacyproto.URLMetadataRequest{
			NetworkName: "freeworld",
			URLs:        []yacymodel.URLHash{first.Hash, second.Hash},
		},
	)

	if got := addressesOf(served); len(got) != 1 || got[0] != first.Address {
		t.Fatalf("served = %v, want only the first asked url", got)
	}
}

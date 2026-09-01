package nodestatus_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type queryRuntimeStatus struct{}

func (queryRuntimeStatus) Version(context.Context) string { return "1.0" }

func (queryRuntimeStatus) Uptime(context.Context) int { return 0 }

func queryIdentity() nodeidentity.Identity {
	return nodeidentity.Identity{
		Hash:        yacymodel.WordHash("self"),
		NetworkName: "freeworld",
	}
}

func muxWithQuery(t *testing.T, counts stubCounter) *http.ServeMux {
	t.Helper()

	mux := http.NewServeMux()
	router := httpguard.NewWireRouter(mux, httpguard.WireGate{
		Guard: httpguard.NewRequestGuard(
			httpguard.DefaultMaxBodyBytes,
			httpguard.DefaultRequestTimeout,
		),
		Respond: httpguard.NewWireResponder(queryRuntimeStatus{}),
		Address: httpguard.NewClientAddressResolver(nil),
	})
	nodestatus.MountQuery(router, queryIdentity(), openVault(t), counts, counts, counts)

	return mux
}

func serveQuery(
	t *testing.T,
	mux *http.ServeMux,
	req yacyproto.QueryRequest,
) yacyproto.QueryResponse {
	t.Helper()

	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		yacyproto.PathQuery,
		strings.NewReader(req.Form().Encode()),
	)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	mux.ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %q", rec.Code, rec.Body.String())
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	resp, err := yacyproto.ParseQueryResponse(yacyproto.ParseMessage(string(body)))
	if err != nil {
		t.Fatalf("ParseQueryResponse: %v", err)
	}

	return resp
}

func queryRequest(object yacyproto.QueryObject) yacyproto.QueryRequest {
	return yacyproto.QueryRequest{
		NetworkName: "freeworld",
		YouAre:      yacymodel.WordHash("self"),
		Iam:         yacymodel.WordHash("caller"),
		Object:      object,
	}
}

func TestQueryAnswersSupportedObjects(t *testing.T) {
	mux := muxWithQuery(t, stubCounter{rwi: 11, refs: 4, urls: 6})

	cases := []struct {
		object yacyproto.QueryObject
		want   int
	}{
		{yacyproto.ObjectRWICount, 11},
		{yacyproto.ObjectRWIURLCount, 4},
		{yacyproto.ObjectLURLCount, 6},
	}
	for _, c := range cases {
		resp := serveQuery(t, mux, queryRequest(c.object))
		if resp.Response != c.want {
			t.Fatalf("%s: Response = %d, want %d", c.object, resp.Response, c.want)
		}
	}
}

func TestQueryRejectsUnsupportedObject(t *testing.T) {
	mux := muxWithQuery(t, stubCounter{rwi: 11})

	resp := serveQuery(t, mux, queryRequest(yacyproto.ObjectWantedSeeds))
	if resp.Response != yacyproto.QueryResponseRejected {
		t.Fatalf("Response = %d, want rejected", resp.Response)
	}
}

func TestQueryRejectsWrongTarget(t *testing.T) {
	mux := muxWithQuery(t, stubCounter{rwi: 11})

	req := queryRequest(yacyproto.ObjectRWICount)
	req.YouAre = yacymodel.WordHash("other")
	resp := serveQuery(t, mux, req)

	if resp.Response != yacyproto.QueryResponseRejected {
		t.Fatalf("Response = %d, want rejected for wrong target", resp.Response)
	}
}

func TestQueryFailsOnCountError(t *testing.T) {
	mux := muxWithQuery(t, stubCounter{err: errors.New("boom")})

	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		yacyproto.PathQuery,
		strings.NewReader(queryRequest(yacyproto.ObjectRWICount).Form().Encode()),
	)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	mux.ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 on count failure", rec.Code)
	}
}

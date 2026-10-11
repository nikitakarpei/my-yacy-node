package nodestatus_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type queryRuntimeStatus struct{}

func (queryRuntimeStatus) Version(context.Context) string { return "1.0" }

func (queryRuntimeStatus) Uptime(context.Context) int { return 0 }

var (
	queryNodeStart  = time.Date(2026, 10, 11, 6, 0, 0, 0, time.UTC)
	queryAnsweredAt = queryNodeStart.Add(90 * time.Minute)
)

func queryIdentity() nodeidentity.Identity {
	return nodeidentity.Identity{
		Hash:        yacymodel.WordHash("self"),
		NetworkName: "freeworld",
		Start:       queryNodeStart,
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
	nodestatus.MountQuery(
		router,
		queryIdentity(),
		clockAt(queryAnsweredAt),
		openVault(t),
		counts,
		counts,
		counts,
	)

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

func TestQueryAnswersNodeCounts(t *testing.T) {
	mux := muxWithQuery(t, stubCounter{rwi: 11, urls: 6})

	cases := []struct {
		object yacyproto.QueryObject
		want   int
	}{
		{yacyproto.ObjectRWICount, 11},
		{yacyproto.ObjectLURLCount, 6},
	}
	for _, c := range cases {
		resp := serveQuery(t, mux, queryRequest(c.object))
		if resp.Response != c.want {
			t.Fatalf("%s: Response = %d, want %d", c.object, resp.Response, c.want)
		}
	}
}

func TestQueryAnswersPostingAmountOfTheWordInEnv(t *testing.T) {
	word := yacymodel.WordHash("linux")
	mux := muxWithQuery(t, stubCounter{postingAmounts: map[yacymodel.Hash]int{word: 4}})

	cases := []struct {
		env  string
		want int
	}{
		{word.String(), 4},
		{yacymodel.WordHash("absent").String(), 0},
		{"", 0},
		{"not-a-hash", 0},
	}
	for _, c := range cases {
		req := queryRequest(yacyproto.ObjectRWIURLCount)
		req.Env = c.env
		resp := serveQuery(t, mux, req)
		if resp.Response != c.want {
			t.Fatalf("env %q: Response = %d, want %d", c.env, resp.Response, c.want)
		}
	}
}

func TestQueryAnswersZeroForWantedObjects(t *testing.T) {
	mux := muxWithQuery(t, stubCounter{rwi: 11})

	for _, object := range []yacyproto.QueryObject{
		yacyproto.ObjectWantedLURLs,
		yacyproto.ObjectWantedPURLs,
		yacyproto.ObjectWantedWord,
		yacyproto.ObjectWantedRWI,
		yacyproto.ObjectWantedSeeds,
	} {
		if resp := serveQuery(t, mux, queryRequest(object)); resp.Response != 0 {
			t.Fatalf("%s: Response = %d, want 0", object, resp.Response)
		}
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

func TestQueryRepliesCarryTimeAndMagic(t *testing.T) {
	mux := muxWithQuery(t, stubCounter{rwi: 11})
	wrongTarget := queryRequest(yacyproto.ObjectRWICount)
	wrongTarget.YouAre = yacymodel.WordHash("other")
	wantMagic := strconv.FormatInt(queryNodeStart.UnixMilli(), 10)

	for _, req := range []yacyproto.QueryRequest{
		queryRequest(yacyproto.ObjectRWICount),
		wrongTarget,
	} {
		resp := serveQuery(t, mux, req)
		if !resp.MyTime.Equal(queryAnsweredAt) {
			t.Fatalf("youare %s: MyTime = %v, want %v", req.YouAre, resp.MyTime, queryAnsweredAt)
		}
		if resp.Magic != wantMagic {
			t.Fatalf("youare %s: Magic = %q, want %q", req.YouAre, resp.Magic, wantMagic)
		}
	}
}

func TestQueryFailsOnCountError(t *testing.T) {
	mux := muxWithQuery(t, stubCounter{err: errors.New("boom")})
	wordCount := queryRequest(yacyproto.ObjectRWIURLCount)
	wordCount.Env = yacymodel.WordHash("linux").String()

	for _, req := range []yacyproto.QueryRequest{
		queryRequest(yacyproto.ObjectRWICount),
		wordCount,
		queryRequest(yacyproto.ObjectLURLCount),
	} {
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			yacyproto.PathQuery,
			strings.NewReader(req.Form().Encode()),
		)
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		mux.ServeHTTP(rec, httpReq)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("%s: status = %d, want 500 on count failure", req.Object, rec.Code)
		}
	}
}

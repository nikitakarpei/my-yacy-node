package messageadmission_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/httpguard"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/messageadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	networkName   = "freeworld"
	senderAddress = "203.0.113.7"
	bodySizeLimit = 10240
)

var (
	selfHash   = yacymodel.WordHash("self")
	senderHash = yacymodel.WordHash("zeus")
)

type statusStub struct{}

func (statusStub) Version(context.Context) string { return "1.0" }

func (statusStub) Uptime(context.Context) int { return 0 }

type inboxStub struct {
	delivers bool
	messages []messageadmission.Message
}

func (i *inboxStub) Deliver(_ context.Context, message messageadmission.Message) bool {
	i.messages = append(i.messages, message)

	return i.delivers
}

type clockStub struct{ instant time.Time }

func (c *clockStub) now() time.Time { return c.instant }

type acceptingNode struct {
	mux   *http.ServeMux
	inbox *inboxStub
	clock *clockStub
}

func newAcceptingNode(t *testing.T) acceptingNode {
	t.Helper()

	node := acceptingNode{
		mux:   http.NewServeMux(),
		inbox: &inboxStub{delivers: true},
		clock: &clockStub{instant: time.Date(2026, time.October, 14, 12, 0, 0, 0, time.UTC)},
	}
	messageadmission.MountAccepting(
		routerOn(node.mux),
		nodeidentity.Identity{Hash: selfHash, NetworkName: networkName},
		node.clock.now,
		node.inbox,
	)

	return node
}

func routerOn(mux *http.ServeMux) httpguard.WireRouter {
	return httpguard.NewWireRouter(mux, httpguard.WireGate{
		Guard: httpguard.NewRequestGuard(
			httpguard.DefaultMaxBodyBytes,
			httpguard.DefaultRequestTimeout,
		),
		Respond: httpguard.NewWireResponder(statusStub{}),
		Address: httpguard.NewClientAddressResolver(nil),
	})
}

func senderSeed(t *testing.T, hash yacymodel.Hash) yacymodel.Seed {
	t.Helper()

	name, err := yacymodel.ParsePeerName("zeus")
	if err != nil {
		t.Fatalf("parse peer name: %v", err)
	}

	return yacymodel.Seed{
		Hash:     hash,
		Name:     name,
		PeerType: yacymodel.PeerSenior,
		Tags:     yacymodel.MatchAllTags(),
	}
}

func postRequest(t *testing.T, subject, body string) yacyproto.MessageRequest {
	t.Helper()

	return yacyproto.MessageRequest{
		NetworkName: networkName,
		YouAre:      selfHash,
		Iam:         senderHash,
		Process:     yacyproto.ProcessPost,
		MySeed:      yacymodel.Some(senderSeed(t, senderHash)),
		Subject:     subject,
		Body:        body,
	}
}

func permissionRequest() yacyproto.MessageRequest {
	return yacyproto.MessageRequest{
		NetworkName: networkName,
		YouAre:      selfHash,
		Iam:         senderHash,
		Process:     yacyproto.ProcessPermission,
	}
}

func send(
	t *testing.T,
	mux *http.ServeMux,
	address string,
	req yacyproto.MessageRequest,
) yacyproto.MessageResponse {
	t.Helper()

	rec := httptest.NewRecorder()
	httpReq := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		yacyproto.PathMessage,
		strings.NewReader(req.Form().Encode()),
	)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.RemoteAddr = address + ":4000"

	mux.ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %q", rec.Code, rec.Body.String())
	}

	resp, err := yacyproto.ParseMessageResponse(yacyproto.ParseResponseFields(rec.Body.String()))
	if err != nil {
		t.Fatalf("parse message response: %v", err)
	}

	return resp
}

func TestPermissionAnnouncesTheMessageSize(t *testing.T) {
	t.Parallel()

	node := newAcceptingNode(t)

	resp := send(t, node.mux, senderAddress, permissionRequest())

	if resp.Response != yacyproto.MessageResponseWelcome {
		t.Fatalf("response = %q, want %q", resp.Response, yacyproto.MessageResponseWelcome)
	}
	if resp.MessageSize != bodySizeLimit || resp.AttachmentSize != 0 {
		t.Fatalf("sizes = %d/%d, want %d/0", resp.MessageSize, resp.AttachmentSize, bodySizeLimit)
	}
}

func TestMessageForAnotherPeerIsRefused(t *testing.T) {
	t.Parallel()

	node := newAcceptingNode(t)
	req := postRequest(t, "subject", "body")
	req.YouAre = yacymodel.WordHash("other")

	resp := send(t, node.mux, senderAddress, req)

	if resp.Response != yacyproto.MessageResponseRefused || resp.MessageSize != 0 {
		t.Fatalf(
			"response = %q, size %d; want refusal with size 0",
			resp.Response,
			resp.MessageSize,
		)
	}
	if len(node.inbox.messages) != 0 {
		t.Fatalf("delivered %d messages, want none", len(node.inbox.messages))
	}
}

func TestMessageReachesTheInboxWithItsSender(t *testing.T) {
	t.Parallel()

	node := newAcceptingNode(t)

	resp := send(t, node.mux, senderAddress, postRequest(t, " hi ", "your peer sends me spam\n"))

	if resp.Response != yacyproto.MessageResponseAccepted {
		t.Fatalf("response = %q, want %q", resp.Response, yacyproto.MessageResponseAccepted)
	}
	want := messageadmission.Message{
		Subject: "hi",
		Body:    "your peer sends me spam",
		Sender: messageadmission.Sender{
			Hash:    senderHash,
			Name:    "zeus",
			Address: senderAddress,
		},
	}
	if len(node.inbox.messages) != 1 || node.inbox.messages[0] != want {
		t.Fatalf("inbox = %#v, want %#v", node.inbox.messages, want)
	}
}

func TestMessageFailingAdmissionIsRefused(t *testing.T) {
	t.Parallel()

	foreignSeed := postRequest(t, "subject", "body")
	foreignSeed.MySeed = yacymodel.Some(senderSeed(t, yacymodel.WordHash("hera")))
	withoutSeed := postRequest(t, "subject", "body")
	withoutSeed.MySeed = yacymodel.None[yacymodel.Seed]()

	cases := map[string]yacyproto.MessageRequest{
		"seed names another peer": foreignSeed,
		"no seed":                 withoutSeed,
		"blank subject":           postRequest(t, "  ", "body"),
		"blank body":              postRequest(t, "subject", "\n"),
		"subject over 100":        postRequest(t, strings.Repeat("s", 101), "body"),
		"body over the size":      postRequest(t, "subject", strings.Repeat("ü", bodySizeLimit+1)),
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			node := newAcceptingNode(t)

			resp := send(t, node.mux, senderAddress, req)

			if resp.Response != yacyproto.MessageResponseRefused {
				t.Fatalf("response = %q, want refusal", resp.Response)
			}
			if len(node.inbox.messages) != 0 {
				t.Fatalf("delivered %d messages, want none", len(node.inbox.messages))
			}
		})
	}
}

func TestMessageAtTheSizeLimitsIsAccepted(t *testing.T) {
	t.Parallel()

	node := newAcceptingNode(t)
	req := postRequest(t, strings.Repeat("s", 100), strings.Repeat("ü", bodySizeLimit))

	resp := send(t, node.mux, senderAddress, req)

	if resp.Response != yacyproto.MessageResponseAccepted {
		t.Fatalf("response = %q, want %q", resp.Response, yacyproto.MessageResponseAccepted)
	}
}

func TestMessageTheInboxDoesNotTakeIsRefused(t *testing.T) {
	t.Parallel()

	node := newAcceptingNode(t)
	node.inbox.delivers = false

	resp := send(t, node.mux, senderAddress, postRequest(t, "subject", "body"))

	if resp.Response != yacyproto.MessageResponseRefused {
		t.Fatalf("response = %q, want refusal", resp.Response)
	}
}

func TestSenderAddressGetsFiveMessagesAnHour(t *testing.T) {
	t.Parallel()

	node := newAcceptingNode(t)
	for i := range 5 {
		resp := send(t, node.mux, senderAddress, postRequest(t, "subject", "body"))
		if resp.Response != yacyproto.MessageResponseAccepted {
			t.Fatalf("message %d: response = %q, want acceptance", i+1, resp.Response)
		}
	}

	if resp := send(t, node.mux, senderAddress, postRequest(t, "subject", "body")); resp.Response !=
		yacyproto.MessageResponseRefused {
		t.Fatalf("sixth message: response = %q, want refusal", resp.Response)
	}
	if resp := send(
		t,
		node.mux,
		"198.51.100.1",
		postRequest(t, "subject", "body"),
	); resp.Response !=
		yacyproto.MessageResponseAccepted {
		t.Fatalf("other address: response = %q, want acceptance", resp.Response)
	}

	node.clock.instant = node.clock.instant.Add(time.Hour)
	if resp := send(t, node.mux, senderAddress, postRequest(t, "subject", "body")); resp.Response !=
		yacyproto.MessageResponseAccepted {
		t.Fatalf("an hour later: response = %q, want acceptance", resp.Response)
	}
}

func TestNewSenderAddressWaitsWhileTheAllowanceTracksTooMany(t *testing.T) {
	t.Parallel()

	node := newAcceptingNode(t)
	for i := range 1024 {
		send(
			t,
			node.mux,
			"10.0."+strconv.Itoa(i/256)+"."+strconv.Itoa(i%256),
			postRequest(t, "s", "b"),
		)
	}

	if resp := send(t, node.mux, senderAddress, postRequest(t, "subject", "body")); resp.Response !=
		yacyproto.MessageResponseRefused {
		t.Fatalf("response = %q, want refusal while full", resp.Response)
	}

	node.clock.instant = node.clock.instant.Add(time.Hour)
	if resp := send(t, node.mux, senderAddress, postRequest(t, "subject", "body")); resp.Response !=
		yacyproto.MessageResponseAccepted {
		t.Fatalf("an hour later: response = %q, want acceptance", resp.Response)
	}
}

func TestNodeWithoutInboxRefusesEveryMessage(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	messageadmission.MountRefusing(routerOn(mux))

	for _, req := range []yacyproto.MessageRequest{permissionRequest(), postRequest(t, "subject", "body")} {
		resp := send(t, mux, senderAddress, req)
		if resp.Response != yacyproto.MessageResponseRefused || resp.MessageSize != 0 {
			t.Fatalf(
				"%s: response = %q, size %d; want refusal with size 0",
				req.Process,
				resp.Response,
				resp.MessageSize,
			)
		}
	}
}

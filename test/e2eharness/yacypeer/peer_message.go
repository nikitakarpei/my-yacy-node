//go:build e2e

package yacypeer

import (
	"context"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

// OpenMessageForm opens the page where the YaCy operator writes a message to
// the peer, and returns the page; YaCy asks the peer for permission to open it.
func OpenMessageForm(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL string,
	peer yacymodel.Hash,
) string {
	t.Helper()

	return postMessagePage(t, ctx, probe, yacyURL, url.Values{"hash": {peer.String()}})
}

// SendMessage sends a message to the peer as the YaCy operator does, and
// returns the page that shows the answer of the peer.
func SendMessage(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL string,
	peer yacymodel.Hash,
	subject, body string,
) string {
	t.Helper()

	return postMessagePage(t, ctx, probe, yacyURL, url.Values{
		"hash":        {peer.String()},
		"subject":     {subject},
		"message":     {body},
		"messagesize": {"10240"},
	})
}

func postMessagePage(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	yacyURL string,
	form url.Values,
) string {
	t.Helper()

	result := probe.PostRaw(ctx, yacyURL+"/MessageSend_p.html", form.Encode(),
		"Content-Type: application/x-www-form-urlencoded", adminAuthHeader)
	if !result.OK {
		t.Fatalf("MessageSend_p.html request to YaCy failed: %s", result.Diag())
	}

	return result.Body
}

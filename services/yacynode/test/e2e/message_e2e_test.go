//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/hermeticnetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/messagewebhook"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/nodepeer"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/peerclient"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/yacypeer"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const webhookAlias = "webhook-e2e"

func TestRealYaCySendsAMessageToTheOperatorWebhook(t *testing.T) {
	ctx := context.Background()
	probe := httpprobe.New(t)

	network := hermeticnetwork.New(t, ctx)

	egressproxy.Start(t, ctx, network.Name)

	_, yacyURL := yacypeer.Start(t, ctx, probe, network.Name, yacyAlias)
	webhook := messagewebhook.Start(t, ctx, network.Name, webhookAlias)

	nodepeer.Start(t, ctx, probe, nodepeer.Config{
		NetworkName:       network.Name,
		Alias:             nodeAlias,
		Hash:              nodeHash,
		SeedlistURL:       "http://" + yacyAlias + ":" + peerclient.Port + "/yacy/seedlist.html",
		MessageWebhookURL: webhook.NetworkURL,
	})

	waitPeerActiveConnected(t, ctx, probe, yacyURL, nodeHash, 15*time.Second)
	yacyHash := peerclient.ResolveHash(t, ctx, probe, yacyURL)

	if form := yacypeer.OpenMessageForm(t, ctx, probe, yacyURL, nodeHash); !strings.Contains(
		form, yacyproto.MessageResponseWelcome,
	) {
		t.Fatalf("YaCy got no permission from the node to write a message:\n%s", form)
	}

	if answer := yacypeer.SendMessage(
		t, ctx, probe, yacyURL, nodeHash, "hi", "your peer sends me spam",
	); !strings.Contains(answer, yacyproto.MessageResponseAccepted) {
		t.Fatalf("the node did not accept the message of YaCy:\n%s", answer)
	}

	messages := webhook.Messages(t, ctx, probe)
	if len(messages) != 1 {
		t.Fatalf("webhook got %d messages, want 1: %#v", len(messages), messages)
	}
	if messages[0].Subject != "hi" || messages[0].Body != "your peer sends me spam" {
		t.Fatalf("webhook got %#v, want the subject and text YaCy sent", messages[0])
	}
	if messages[0].Sender.Hash != yacyHash.String() || messages[0].Sender.Address == "" {
		t.Fatalf(
			"webhook got sender %#v, want YaCy %s with its address",
			messages[0].Sender,
			yacyHash,
		)
	}
}

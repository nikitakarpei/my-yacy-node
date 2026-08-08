package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerepresentations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/recallclienttest"
)

const (
	serveTimeLimit = 10 * time.Second
	retryPause     = 50 * time.Millisecond
)

func TestAServedReceiverAnswersRecallsUntilTheContextEnds(t *testing.T) {
	listenAddress := freeListenAddress(t)
	receiver := markdownReceiverAt(t, &fakeRecaller{result: recall.RecalledPage{
		Representations: []recall.RecalledRepresentation{{
			Kind:           markdown.Kind,
			Representation: markdown.Page{CanonicalURL: recalledURL, Markdown: "# Hi"},
		}},
	}}, listenAddress)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	served := make(chan error, 1)
	go func() { served <- receiver.Serve(ctx) }()

	response := recallOverTheContract(t, listenAddress)
	if response.GetRepresentations()[0].GetMarkdown().GetMarkdown() != "# Hi" {
		t.Errorf("representations = %v", response.GetRepresentations())
	}

	cancel()
	select {
	case err := <-served:
		if err != nil {
			t.Errorf("serve: %v", err)
		}
	case <-time.After(serveTimeLimit):
		t.Fatal("receiver did not stop after cancel")
	}
}

func freeListenAddress(t *testing.T) string {
	t.Helper()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve address: %v", err)
	}
	listenAddress := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release address: %v", err)
	}
	return listenAddress
}

func markdownReceiverAt(
	t *testing.T,
	recaller Recaller,
	listenAddress string,
) *RecallReceiver {
	t.Helper()
	receiver, err := NewRecallReceiver(
		recaller, markdownCorpora(), listenAddress,
	)
	if err != nil {
		t.Fatalf("new recall receiver: %v", err)
	}
	return receiver
}

func recallOverTheContract(t *testing.T, listenAddress string) *corpusrecallv1.RecallResponse {
	t.Helper()
	client := recallclienttest.New(t, listenAddress)
	deadline := time.Now().Add(serveTimeLimit)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), serveTimeLimit)
		response, err := client.Recall(ctx, &corpusrecallv1.RecallRequest{Url: recalledURL})
		cancel()
		if err == nil {
			return response
		}
		time.Sleep(retryPause)
	}
	t.Fatal("recall never reached the served receiver")
	return nil
}

func TestAReceiverRefusesToServeAListenAddressThatCannotBind(t *testing.T) {
	receiver := markdownReceiverAt(t, &fakeRecaller{}, "127.0.0.1:99999")

	if err := receiver.Serve(context.Background()); err == nil {
		t.Fatal("expected error when listen address cannot bind")
	}
}

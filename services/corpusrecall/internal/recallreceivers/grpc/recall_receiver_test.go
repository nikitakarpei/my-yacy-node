package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerepresentations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recallreceivers/grpc"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/recallclienttest"
)

const (
	serveTimeLimit = 10 * time.Second
	retryPause     = 50 * time.Millisecond
)

type receiverUnderTest struct {
	t      *testing.T
	client corpusrecallv1.RecallClient
}

func recallReceiverUnderTest(t *testing.T, recaller grpc.Recaller) receiverUnderTest {
	t.Helper()
	listenAddress := freeListenAddress(t)
	receiver, err := grpc.NewRecallReceiver(recaller, markdownCorpora(), listenAddress)
	if err != nil {
		t.Fatalf("new recall receiver: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- receiver.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-served:
			if err != nil {
				t.Errorf("serve: %v", err)
			}
		case <-time.After(serveTimeLimit):
			t.Error("receiver did not stop after cancel")
		}
	})
	waitUntilListening(t, listenAddress)
	return receiverUnderTest{t: t, client: recallclienttest.New(t, listenAddress)}
}

func (r receiverUnderTest) Recall(
	request *corpusrecallv1.RecallRequest,
) (*corpusrecallv1.RecallResponse, error) {
	r.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), serveTimeLimit)
	defer cancel()
	return r.client.Recall(ctx, request)
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

func waitUntilListening(t *testing.T, listenAddress string) {
	t.Helper()
	var dialer net.Dialer
	deadline := time.Now().Add(serveTimeLimit)
	for time.Now().Before(deadline) {
		connection, err := dialer.DialContext(context.Background(), "tcp", listenAddress)
		if err == nil {
			_ = connection.Close()
			return
		}
		time.Sleep(retryPause)
	}
	t.Fatalf("receiver never listened at %s", listenAddress)
}

const kindWithoutForm recall.RepresentationKind = "text"

type fakeCorpus struct {
	kind recall.RepresentationKind
}

func (c fakeCorpus) RepresentationKind() recall.RepresentationKind { return c.kind }

func (fakeCorpus) RepresentationOf(
	_ context.Context,
	_ string,
) (recall.Representation, bool, error) {
	return nil, false, nil
}

func markdownCorpora() []recall.Corpus {
	return []recall.Corpus{fakeCorpus{kind: markdown.Kind}}
}

func TestAReceiverRefusesToServeAListenAddressThatCannotBind(t *testing.T) {
	receiver, err := grpc.NewRecallReceiver(
		&fakeRecaller{}, markdownCorpora(), "127.0.0.1:99999",
	)
	if err != nil {
		t.Fatalf("new recall receiver: %v", err)
	}

	if err := receiver.Serve(context.Background()); err == nil {
		t.Fatal("expected error when listen address cannot bind")
	}
}

func TestAReceiverRefusesACorpusWhoseKindHasNoContractForm(t *testing.T) {
	_, err := grpc.NewRecallReceiver(
		&fakeRecaller{},
		[]recall.Corpus{fakeCorpus{kind: kindWithoutForm}},
		freeListenAddress(t),
	)

	if !errors.Is(err, grpc.ErrRepresentationKindNotInContract) {
		t.Fatalf("error = %v, want %v", err, grpc.ErrRepresentationKindNotInContract)
	}
}

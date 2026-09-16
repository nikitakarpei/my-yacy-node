// Package peerlivenesswire asks one peer address whether it is alive. An
// address is alive when it answers the RWI count query as a peer of this
// network, so an address that answers something else is not mistaken for a
// peer.
package peerlivenesswire

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	livenessPath      = "/yacy/query.html"
	answerByteCeiling = 64 << 10
)

var (
	errAnswerCarriedNoRWICount = errors.New("the answer carried no RWI count")
	errRWICountRejected        = errors.New("the address rejected the RWI count query")
)

type PeerLivenessObserver interface {
	ProbeCouldNotBeBuilt(ctx context.Context, address string, err error)
	PeerDidNotAnswerTheProbe(ctx context.Context, address string, err error)
	PeerRefusedTheProbe(ctx context.Context, address string, status int)
	ProbeAnswerCouldNotBeRead(ctx context.Context, address string, err error)
	ProbeAnswerCarriedNoRWICount(ctx context.Context, address string, err error)
}

type Wire struct {
	client      *http.Client
	networkName string
	observer    PeerLivenessObserver
}

func New(client *http.Client, networkName string, observer PeerLivenessObserver) Wire {
	return Wire{client: client, networkName: networkName, observer: observer}
}

func (w Wire) Alive(ctx context.Context, address string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.probeOf(address), nil)
	if err != nil {
		w.observer.ProbeCouldNotBeBuilt(ctx, address, err)

		return false
	}
	resp, err := w.client.Do(req)
	if err != nil {
		w.observer.PeerDidNotAnswerTheProbe(ctx, address, err)

		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return w.answersAsAPeer(ctx, address, resp)
}

func (w Wire) probeOf(address string) string {
	return address + livenessPath + "?" + url.Values{
		yacyproto.FieldNetworkName: {w.networkName},
		yacyproto.FieldObject:      {string(yacyproto.ObjectRWICount)},
	}.Encode()
}

func (w Wire) answersAsAPeer(ctx context.Context, address string, resp *http.Response) bool {
	if resp.StatusCode != http.StatusOK {
		w.observer.PeerRefusedTheProbe(ctx, address, resp.StatusCode)

		return false
	}
	answer, err := io.ReadAll(io.LimitReader(resp.Body, answerByteCeiling))
	if err != nil {
		w.observer.ProbeAnswerCouldNotBeRead(ctx, address, err)

		return false
	}
	if err := rwiCountFailureIn(answer); err != nil {
		w.observer.ProbeAnswerCarriedNoRWICount(ctx, address, err)

		return false
	}

	return true
}

func rwiCountFailureIn(answer []byte) error {
	message := yacyproto.ParseMessage(string(answer))
	if _, carried := message[yacyproto.FieldResponse]; !carried {
		return errAnswerCarriedNoRWICount
	}
	rwiCountAnswer, err := yacyproto.ParseQueryResponse(message)
	if err != nil {
		return err
	}
	if rwiCountAnswer.Response == yacyproto.QueryResponseRejected {
		return errRWICountRejected
	}

	return nil
}

type PeerLivenessObservers []PeerLivenessObserver

func (observers PeerLivenessObservers) ProbeCouldNotBeBuilt(
	ctx context.Context,
	address string,
	err error,
) {
	for _, observer := range observers {
		observer.ProbeCouldNotBeBuilt(ctx, address, err)
	}
}

func (observers PeerLivenessObservers) PeerDidNotAnswerTheProbe(
	ctx context.Context,
	address string,
	err error,
) {
	for _, observer := range observers {
		observer.PeerDidNotAnswerTheProbe(ctx, address, err)
	}
}

func (observers PeerLivenessObservers) PeerRefusedTheProbe(
	ctx context.Context,
	address string,
	status int,
) {
	for _, observer := range observers {
		observer.PeerRefusedTheProbe(ctx, address, status)
	}
}

func (observers PeerLivenessObservers) ProbeAnswerCouldNotBeRead(
	ctx context.Context,
	address string,
	err error,
) {
	for _, observer := range observers {
		observer.ProbeAnswerCouldNotBeRead(ctx, address, err)
	}
}

func (observers PeerLivenessObservers) ProbeAnswerCarriedNoRWICount(
	ctx context.Context,
	address string,
	err error,
) {
	for _, observer := range observers {
		observer.ProbeAnswerCarriedNoRWICount(ctx, address, err)
	}
}

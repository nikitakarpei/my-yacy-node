// Package yggdrasilpeerlist reads a public Yggdrasil peer list, in the JSON format of
// publicpeers.neilalexander.dev. It gives the peers that the list states are up and speak the
// protocol of the node, each with the public key that the list states for it.
package yggdrasilpeerlist

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/yggdrasil-network/yggdrasil-go/src/core"
)

const (
	readTimeout = 30 * time.Second
	listCeiling = 4 << 20
)

var errListRefused = errors.New("the list server refused the list")

var connectableSchemes = map[string]bool{"tcp": true, "tls": true}

type Peer struct {
	URL *url.URL
	Key ed25519.PublicKey
}

type listedPeer struct {
	Up                   bool   `json:"up"`
	Key                  string `json:"key"`
	ProtocolMinorVersion uint16 `json:"proto_minor"`
}

type List struct {
	listURL *url.URL
	client  *http.Client
}

func New(listURL *url.URL) *List {
	return &List{listURL: listURL, client: &http.Client{Timeout: readTimeout}}
}

func (l *List) UpPeers(ctx context.Context) []Peer {
	regions, err := l.regions(ctx)
	if err != nil {
		l.reportUnread(ctx, err)
		return nil
	}
	upPeers := upPeersIn(ctx, regions)
	l.reportRead(ctx, upPeers)
	return upPeers
}

func (l *List) regions(ctx context.Context) (map[string]map[string]listedPeer, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, l.listURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build the request: %w", err)
	}
	response, err := l.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request the list: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", errListRefused, response.Status)
	}
	var regions map[string]map[string]listedPeer
	if err := json.NewDecoder(io.LimitReader(response.Body, listCeiling)).
		Decode(&regions); err != nil {
		return nil, fmt.Errorf("decode the list: %w", err)
	}
	return regions, nil
}

func (l *List) reportUnread(ctx context.Context, err error) {
	slog.WarnContext(ctx, "public Yggdrasil peer list not read",
		slog.String("list", l.listURL.Redacted()),
		slog.Any("reason", err),
	)
}

func upPeersIn(ctx context.Context, regions map[string]map[string]listedPeer) []Peer {
	var upPeers []Peer
	for _, region := range regions {
		for listedURL, listed := range region {
			if peer, isUp := upPeerFrom(ctx, listedURL, listed); isUp {
				upPeers = append(upPeers, peer)
			}
		}
	}
	slices.SortFunc(upPeers, func(a, b Peer) int {
		return strings.Compare(a.URL.String(), b.URL.String())
	})
	return upPeers
}

func upPeerFrom(ctx context.Context, listedURL string, listed listedPeer) (Peer, bool) {
	if !listed.Up || listed.ProtocolMinorVersion != core.ProtocolVersionMinor {
		return Peer{}, false
	}
	peerURL, urlErr := url.Parse(listedURL)
	key, keyErr := hex.DecodeString(listed.Key)
	if urlErr != nil || keyErr != nil || len(key) != ed25519.PublicKeySize {
		slog.WarnContext(ctx, "public Yggdrasil peer not readable",
			slog.String("peer", listedURL),
			slog.String("key", listed.Key),
		)
		return Peer{}, false
	}
	if !connectableSchemes[peerURL.Scheme] {
		return Peer{}, false
	}
	return Peer{URL: peerURL, Key: key}, true
}

func (l *List) reportRead(ctx context.Context, upPeers []Peer) {
	slog.DebugContext(ctx, "public Yggdrasil peer list read",
		slog.String("list", l.listURL.Redacted()),
		slog.Int("upPeers", len(upPeers)),
	)
}

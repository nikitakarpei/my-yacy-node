// Package webhook gives each admitted message to the operator's webhook as one
// JSON POST. A webhook whose host resolves only to private addresses is called
// directly on the addresses resolved for that delivery. Any other webhook is
// called over HTTPS through the egress proxy.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/messageadmission"
)

const (
	deliveryTimeout       = 10 * time.Second
	responseDrainLimit    = 4096
	directDialNetwork     = "tcp"
	resolvedAddressFamily = "ip"
)

var sharedAddressSpace = netip.MustParsePrefix("100.64.0.0/10")

var errRedirectRefused = errors.New("webhook redirect refused")

type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

type Inbox struct {
	target   *url.URL
	resolver Resolver
	proxied  *http.Client
}

func New(target, egressProxy *url.URL, resolver Resolver) Inbox {
	return Inbox{
		target:   target,
		resolver: resolver,
		proxied: &http.Client{
			Transport:     &http.Transport{Proxy: http.ProxyURL(egressProxy)},
			CheckRedirect: refuseRedirect,
		},
	}
}

func (i Inbox) Deliver(ctx context.Context, message messageadmission.Message) bool {
	ctx, cancel := context.WithTimeout(ctx, deliveryTimeout)
	defer cancel()

	client, routed := i.clientFor(ctx)
	if !routed {
		return false
	}

	return i.post(ctx, client, message)
}

func (i Inbox) clientFor(ctx context.Context) (*http.Client, bool) {
	addresses, err := i.resolver.LookupNetIP(ctx, resolvedAddressFamily, i.target.Hostname())
	if err != nil || len(addresses) == 0 {
		slog.WarnContext(ctx, "message webhook host not resolved",
			slog.String("host", i.target.Hostname()),
			slog.Any("error", err),
		)

		return nil, false
	}

	if allPrivate(addresses) {
		return directClientTo(addresses), true
	}

	if i.target.Scheme != "https" {
		slog.ErrorContext(ctx, "message webhook on a public address needs https",
			slog.String("host", i.target.Hostname()),
		)

		return nil, false
	}

	return i.proxied, true
}

func allPrivate(addresses []netip.Addr) bool {
	for _, address := range addresses {
		if !isPrivate(address.Unmap()) {
			return false
		}
	}

	return true
}

func isPrivate(address netip.Addr) bool {
	return address.IsLoopback() ||
		address.IsPrivate() ||
		address.IsLinkLocalUnicast() ||
		sharedAddressSpace.Contains(address)
}

func directClientTo(addresses []netip.Addr) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, hostPort string) (net.Conn, error) {
				return dialFirstOf(
					ctx,
					addresses,
					hostPort[strings.LastIndexByte(hostPort, ':')+1:],
				)
			},
			DisableKeepAlives: true,
		},
		CheckRedirect: refuseRedirect,
	}
}

func dialFirstOf(ctx context.Context, addresses []netip.Addr, port string) (net.Conn, error) {
	var dialer net.Dialer
	var failures []error
	for _, address := range addresses {
		conn, err := dialer.DialContext(
			ctx,
			directDialNetwork,
			net.JoinHostPort(address.String(), port),
		)
		if err == nil {
			return conn, nil
		}
		failures = append(failures, err)
	}

	return nil, errors.Join(failures...)
}

func refuseRedirect(*http.Request, []*http.Request) error {
	return errRedirectRefused
}

func (i Inbox) post(
	ctx context.Context,
	client *http.Client,
	message messageadmission.Message,
) bool {
	payload, _ := json.Marshal(payloadOf(message)) //nolint:errchkjson // strings always encode

	req := (&http.Request{
		Method:        http.MethodPost,
		URL:           i.target,
		Header:        http.Header{"Content-Type": {"application/json"}},
		Body:          io.NopCloser(bytes.NewReader(payload)),
		ContentLength: int64(len(payload)),
	}).WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		slog.WarnContext(ctx, "message webhook not reached",
			slog.String("host", i.target.Hostname()),
			slog.Any("error", errors.Unwrap(err)),
		)

		return false
	}
	defer closeResponseBody(ctx, resp.Body)
	_, _ = io.CopyN(io.Discard, resp.Body, responseDrainLimit)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		slog.WarnContext(ctx, "message webhook refused the message",
			slog.String("host", i.target.Hostname()),
			slog.Int("status", resp.StatusCode),
		)

		return false
	}

	return true
}

type messagePayload struct {
	Subject string        `json:"subject"`
	Body    string        `json:"body"`
	Sender  senderPayload `json:"sender"`
}

type senderPayload struct {
	Hash    string `json:"hash"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

func payloadOf(message messageadmission.Message) messagePayload {
	return messagePayload{
		Subject: message.Subject,
		Body:    message.Body,
		Sender: senderPayload{
			Hash:    message.Sender.Hash.String(),
			Name:    message.Sender.Name,
			Address: message.Sender.Address,
		},
	}
}

func closeResponseBody(ctx context.Context, body io.Closer) {
	if err := body.Close(); err != nil {
		slog.DebugContext(ctx, "message webhook response not closed", slog.Any("error", err))
	}
}

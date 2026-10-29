// Package egresstunnel opens a TCP connection to an address through the egress proxy with HTTP
// CONNECT. It talks TLS to an https proxy and checks the certificate of the proxy.
package egresstunnel

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
)

var (
	errTunnelRefused   = errors.New("proxy refused the tunnel")
	errTunnelWithBody  = errors.New("proxy answered the tunnel with a body")
	errEarlyProxyBytes = errors.New("proxy sent bytes before the tunnel opened")
)

var proxyDefaultPorts = map[string]string{"http": "80", "https": "443"}

func ConnectionThrough(ctx context.Context, proxyURL *url.URL, address string) (net.Conn, error) {
	proxyConnection, err := proxyConnectionTo(ctx, proxyURL)
	if err != nil {
		return nil, err
	}
	if err := openTunnel(ctx, proxyConnection, proxyURL, address); err != nil {
		return nil, err
	}

	return proxyConnection, nil
}

func proxyConnectionTo(ctx context.Context, proxyURL *url.URL) (net.Conn, error) {
	var dialer net.Dialer
	connection, err := dialer.DialContext(ctx, "tcp", proxyAddressOf(proxyURL))
	if err != nil {
		return nil, fmt.Errorf("dial the proxy: %w", err)
	}
	if proxyURL.Scheme != "https" {
		return connection, nil
	}
	tlsConnection := tls.Client(connection, &tls.Config{
		ServerName: proxyURL.Hostname(),
		MinVersion: tls.VersionTLS12,
	})
	if err := tlsConnection.HandshakeContext(ctx); err != nil {
		return nil, errors.Join(
			fmt.Errorf("TLS handshake with the proxy: %w", err),
			connection.Close(),
		)
	}

	return tlsConnection, nil
}

func proxyAddressOf(proxyURL *url.URL) string {
	if proxyURL.Port() != "" {
		return proxyURL.Host
	}

	return net.JoinHostPort(proxyURL.Hostname(), proxyDefaultPorts[proxyURL.Scheme])
}

func openTunnel(ctx context.Context, connection net.Conn, proxyURL *url.URL, address string) error {
	stopClosingOnCancel := context.AfterFunc(ctx, func() { _ = connection.Close() })
	err := askForTunnel(connection, proxyURL, address)
	if !stopClosingOnCancel() {
		return fmt.Errorf("open the tunnel: %w", context.Cause(ctx))
	}
	if err != nil {
		_ = connection.Close()

		return err
	}

	return nil
}

func askForTunnel(connection net.Conn, proxyURL *url.URL, address string) error {
	connect := connectRequestFor(proxyURL, address)
	if err := connect.Write(connection); err != nil {
		return fmt.Errorf("ask the proxy for a tunnel: %w", err)
	}
	replies := bufio.NewReader(connection)
	reply, err := http.ReadResponse(replies, connect)
	if err != nil {
		return fmt.Errorf("read the proxy reply: %w", err)
	}
	announcesBody := reply.ContentLength > 0 || len(reply.TransferEncoding) > 0
	if announcesBody {
		_ = connection.Close()
	}
	_ = reply.Body.Close()
	switch {
	case reply.StatusCode != http.StatusOK:
		return fmt.Errorf("%w: %s", errTunnelRefused, reply.Status)
	case announcesBody:
		return errTunnelWithBody
	case replies.Buffered() > 0:
		return errEarlyProxyBytes
	default:
		return nil
	}
}

func connectRequestFor(proxyURL *url.URL, address string) *http.Request {
	connect := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: address},
		Host:   address,
		Header: make(http.Header),
	}
	if proxyURL.User != nil {
		password, _ := proxyURL.User.Password()
		credentials := proxyURL.User.Username() + ":" + password
		connect.Header.Set("Proxy-Authorization",
			"Basic "+base64.StdEncoding.EncodeToString([]byte(credentials)))
	}

	return connect
}

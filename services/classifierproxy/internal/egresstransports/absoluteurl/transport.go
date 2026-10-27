// Package absoluteurl sends each request to the egress proxy in absolute-URL
// form, for https as for http.
package absoluteurl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
)

type Transport struct {
	proxyAddress string
	dialer       net.Dialer
}

func New(proxyURL *url.URL) *Transport {
	return &Transport{proxyAddress: proxyAddressOf(proxyURL)}
}

func proxyAddressOf(proxyURL *url.URL) string {
	if proxyURL.Port() != "" {
		return proxyURL.Host
	}
	return net.JoinHostPort(proxyURL.Hostname(), "80")
}

func (t *Transport) RoundTrip(request *http.Request) (*http.Response, error) {
	connection, err := t.dialer.DialContext(request.Context(), "tcp", t.proxyAddress)
	if err != nil {
		return nil, fmt.Errorf("dial egress proxy: %w", err)
	}
	announceConnection(request.Context(), connection)
	stopClosing := context.AfterFunc(request.Context(), func() { _ = connection.Close() })
	response, err := responseOver(connection, request)
	if err != nil {
		stopClosing()
		_ = connection.Close()
		return nil, err
	}
	response.Body = connectionBody{
		ReadCloser:  response.Body,
		connection:  connection,
		stopClosing: stopClosing,
	}
	return response, nil
}

func announceConnection(ctx context.Context, connection net.Conn) {
	if trace := httptrace.ContextClientTrace(ctx); trace != nil && trace.GotConn != nil {
		trace.GotConn(httptrace.GotConnInfo{Conn: connection})
	}
}

func responseOver(connection net.Conn, request *http.Request) (*http.Response, error) {
	closingRequest := request.Clone(request.Context())
	closingRequest.Close = true
	err := closingRequest.WriteProxy(connection)
	announceWrittenRequest(request.Context(), err)
	if err != nil {
		return nil, fmt.Errorf("write request to egress proxy: %w", err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), closingRequest)
	if err != nil {
		return nil, fmt.Errorf("read response headers of egress proxy: %w", err)
	}
	return response, nil
}

func announceWrittenRequest(ctx context.Context, writeFailure error) {
	if trace := httptrace.ContextClientTrace(ctx); trace != nil && trace.WroteRequest != nil {
		trace.WroteRequest(httptrace.WroteRequestInfo{Err: writeFailure})
	}
}

type connectionBody struct {
	io.ReadCloser
	connection  net.Conn
	stopClosing func() bool
}

func (b connectionBody) Close() error {
	b.stopClosing()
	return b.connection.Close() //nolint:wrapcheck // closing reports nothing the consumer acts on
}

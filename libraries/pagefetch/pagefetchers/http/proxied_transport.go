package http

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
)

func transportForDialMode(proxyURL *url.URL, dialMode ProxyDialMode) http.RoundTripper {
	if dialMode == ProxyDialAbsoluteURL {
		return &absoluteURLTransport{proxyAddr: proxyURL.Host}
	}
	return &http.Transport{Proxy: http.ProxyURL(proxyURL)}
}

// absoluteURLTransport sends the target as an absolute-URI request line to the
// proxy over a plain connection, for proxies that refuse to tunnel via CONNECT.
type absoluteURLTransport struct {
	proxyAddr string
}

func (t *absoluteURLTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	ctx := request.Context()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", t.proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("dial proxy %s: %w", t.proxyAddr, err)
	}
	if deadline, bounded := ctx.Deadline(); bounded {
		_ = conn.SetDeadline(deadline)
	}
	releaseConn := connClosedWhenTheContextEnds(ctx, conn)

	if err := request.WriteProxy(conn); err != nil {
		releaseConn()
		_ = conn.Close()
		return nil, failureOf(ctx, "write request to proxy", t.proxyAddr, err)
	}

	response, err := http.ReadResponse(bufio.NewReader(conn), request)
	if err != nil {
		releaseConn()
		_ = conn.Close()
		return nil, failureOf(ctx, "read response from proxy", t.proxyAddr, err)
	}
	response.Body = &connClosingBody{
		ReadCloser:  response.Body,
		conn:        conn,
		releaseConn: releaseConn,
	}

	return response, nil
}

func connClosedWhenTheContextEnds(ctx context.Context, conn net.Conn) func() {
	released := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-released:
		}
	}()

	var releaseOnce sync.Once

	return func() { releaseOnce.Do(func() { close(released) }) }
}

func failureOf(ctx context.Context, attempt string, proxyAddr string, cause error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("%s %s: %w", attempt, proxyAddr, ctxErr)
	}

	return fmt.Errorf("%s %s: %w", attempt, proxyAddr, cause)
}

type connClosingBody struct {
	io.ReadCloser
	conn        net.Conn
	releaseConn func()
}

func (b *connClosingBody) Close() error {
	b.releaseConn()
	bodyErr := b.ReadCloser.Close()
	connErr := b.conn.Close()
	if bodyErr != nil {
		return fmt.Errorf("close response body: %w", bodyErr)
	}
	if connErr != nil {
		return fmt.Errorf("close proxy connection: %w", connErr)
	}

	return nil
}

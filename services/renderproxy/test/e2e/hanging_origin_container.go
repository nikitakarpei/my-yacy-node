//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
)

const (
	hangingOriginAlias = "hangingorigin"
	hangingOriginImage = "docker.io/library/python:3-alpine"

	// hangingOriginScript answers every connection with a valid 200 header whose
	// Content-Length outruns the bytes it sends, then holds the socket open. The
	// browser observes the main-document status but its load event never fires,
	// reproducing a render that hangs until the deadline cancels it.
	hangingOriginScript = `
import socket
s = socket.socket()
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(("0.0.0.0", 80))
s.listen(50)
held = []
while True:
    c, _ = s.accept()
    c.sendall(b"HTTP/1.1 200 OK\r\nContent-Type: text/html\r\nContent-Length: 1000000\r\n\r\n<html><body>partial")
    held.append(c)
`
)

func startHangingOrigin(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          hangingOriginImage,
			ExposedPorts:   []string{"80/tcp"},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {hangingOriginAlias}},
			Cmd:            []string{"python3", "-c", hangingOriginScript},
			WaitingFor:     wait.ForListeningPort("80/tcp"),
		},
	})
	if err != nil {
		t.Fatalf("start hanging origin container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, hangingOriginAlias, container)
	return "http://" + hangingOriginAlias + "/"
}

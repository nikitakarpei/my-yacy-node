//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
)

const (
	compressedImage        = "docker.io/library/python:3-alpine"
	compressedPath         = "/payload.json"
	compressedPayloadSize  = 200000
	compressedOriginPython = `import gzip, http.server

RAW = b'{"pad":"' + b'a' * %d + b'"}'
GZ = gzip.compress(RAW)

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        gzipped = "gzip" in self.headers.get("Accept-Encoding", "")
        body = GZ if gzipped else RAW
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        if gzipped:
            self.send_header("Content-Encoding", "gzip")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *args):
        pass

http.server.HTTPServer(("0.0.0.0", 80), Handler).serve_forever()
`
)

// startCompressedOrigin serves a payload that compresses to a few hundred bytes and
// unpacks to compressedPayloadSize, so a limit between the two tells the sizes apart.
func startCompressedOrigin(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          compressedImage,
			ExposedPorts:   []string{"80/tcp"},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {originAlias}},
			Files: []testcontainers.ContainerFile{{
				Reader: strings.NewReader(
					fmt.Sprintf(compressedOriginPython, compressedPayloadSize),
				),
				ContainerFilePath: "/origin.py",
				FileMode:          0o644,
			}},
			Cmd:        []string{"python", "/origin.py"},
			WaitingFor: wait.ForHTTP(compressedPath).WithStartupTimeout(time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("start compressed origin container %s: %v", compressedImage, err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, originAlias, container)
	return "http://" + originAlias + compressedPath
}

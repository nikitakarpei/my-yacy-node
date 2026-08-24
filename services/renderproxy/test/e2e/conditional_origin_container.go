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
	conditionalImage        = "docker.io/library/python:3-alpine"
	conditionalEntityTag    = `"page-v1"`
	conditionalCacheControl = "max-age=60"
	conditionalOriginPython = `import http.server

ETAG = '''%s'''
CACHE_CONTROL = '''%s'''
PAGE = b'''%s'''

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        unchanged = self.headers.get("If-None-Match") == ETAG
        self.send_response(304 if unchanged else 200)
        self.send_header("ETag", ETAG)
        self.send_header("Cache-Control", CACHE_CONTROL)
        if unchanged:
            self.end_headers()
            return
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(PAGE)))
        self.end_headers()
        self.wfile.write(PAGE)

    def log_message(self, *args):
        pass

http.server.HTTPServer(("0.0.0.0", 80), Handler).serve_forever()
`
)

// startConditionalOrigin serves a scripted page that states conditionalEntityTag and
// conditionalCacheControl, and answers a request that states that entity tag with 304.
func startConditionalOrigin(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          conditionalImage,
			ExposedPorts:   []string{"80/tcp"},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {originAlias}},
			Files: []testcontainers.ContainerFile{{
				Reader: strings.NewReader(fmt.Sprintf(
					conditionalOriginPython,
					conditionalEntityTag,
					conditionalCacheControl,
					scriptedOriginPage,
				)),
				ContainerFilePath: "/origin.py",
				FileMode:          0o644,
			}},
			Cmd:        []string{"python", "/origin.py"},
			WaitingFor: wait.ForHTTP("/").WithStartupTimeout(time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("start conditional origin container %s: %v", conditionalImage, err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, originAlias, container)
	return "http://" + originAlias + "/"
}

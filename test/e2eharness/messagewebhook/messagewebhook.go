//go:build e2e

// Package messagewebhook starts a webhook that records each message the node
// gives it and lists the recorded messages to the test.
package messagewebhook

import (
	"bufio"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerurl"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
)

const (
	image       = "docker.io/library/python:3-alpine"
	port        = "8080"
	exposedPort = port + "/tcp"
)

const recorder = `
import http.server

class Recorder(http.server.BaseHTTPRequestHandler):
    messages = []

    def do_POST(self):
        body = self.rfile.read(int(self.headers.get("Content-Length", 0)))
        Recorder.messages.append(body.decode())
        self.send_response(200)
        self.end_headers()

    def do_GET(self):
        listed = "\n".join(Recorder.messages).encode()
        self.send_response(200)
        self.send_header("Content-Length", str(len(listed)))
        self.end_headers()
        self.wfile.write(listed)

http.server.HTTPServer(("", ` + port + `), Recorder).serve_forever()
`

type Message struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Sender  Sender `json:"sender"`
}

type Sender struct {
	Hash    string `json:"hash"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

type Webhook struct {
	NetworkURL string
	hostURL    string
}

func Start(t *testing.T, ctx context.Context, networkName, alias string) Webhook {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          image,
			ExposedPorts:   []string{exposedPort},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {alias}},
			Cmd:            []string{"python", "-u", "-c", recorder},
			WaitingFor:     wait.ForListeningPort(exposedPort).WithStartupTimeout(time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("start message webhook container %s: %v", image, err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, alias, container)

	return Webhook{
		NetworkURL: "http://" + alias + ":" + port + "/peer-messages",
		hostURL:    containerurl.HostURL(t, ctx, container, exposedPort),
	}
}

func (w Webhook) Messages(t *testing.T, ctx context.Context, probe *httpprobe.Probe) []Message {
	t.Helper()
	result := probe.Get(ctx, w.hostURL+"/")
	if !result.OK {
		t.Fatalf("list recorded messages: %s", result.Diag())
	}

	var messages []Message
	lines := bufio.NewScanner(strings.NewReader(result.Body))
	for lines.Scan() {
		var message Message
		if err := json.Unmarshal(lines.Bytes(), &message); err != nil {
			t.Fatalf("recorded message %q is not JSON: %v", lines.Text(), err)
		}
		messages = append(messages, message)
	}

	return messages
}

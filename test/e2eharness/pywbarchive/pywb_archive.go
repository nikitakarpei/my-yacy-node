//go:build e2e

// Package pywbarchive starts a seeded frameless pywb archive for end-to-end tests.
package pywbarchive

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerurl"
)

const (
	Image         = "docker.io/webrecorder/pywb:2.9.1"
	Collection    = "captures"
	port          = "8080/tcp"
	alias         = "pywb"
	uncapturedURL = "http://uncaptured.test/"
)

type Archive struct {
	hostURL    string
	networkURL string
}

func Start(t *testing.T, ctx context.Context, networkName string, warc []byte) Archive {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:      Image,
			Entrypoint: []string{"sh"},
			Cmd: []string{
				"-c",
				"wb-manager init " + Collection + " && wb-manager add " + Collection + " /fixtures/archive.warc && exec wayback -b 0.0.0.0 -p 8080",
			},
			Env:            map[string]string{"PYWB_CONFIG_FILE": "/fixtures/config.yaml"},
			ExposedPorts:   []string{port},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {alias}},
			Files: []testcontainers.ContainerFile{
				{
					Reader:            bytes.NewReader(warc),
					ContainerFilePath: "/fixtures/archive.warc",
					FileMode:          0o644,
				},
				{
					Reader: bytes.NewReader(
						[]byte(
							"framed_replay: false\ncollections:\n  " + Collection + ": " + Collection + "\n",
						),
					),
					ContainerFilePath: "/fixtures/config.yaml",
					FileMode:          0o644,
				},
			},
			WaitingFor: wait.ForHTTP("/" + Collection + "/cdx?url=" + uncapturedURL).
				WithPort(port).
				WithForcedIPv4LocalHost().
				WithStartupTimeout(2 * time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("start pywb container %s: %v", Image, err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "pywb", container)
	return Archive{
		hostURL:    containerurl.HostURL(t, ctx, container, port),
		networkURL: "http://" + net.JoinHostPort(alias, "8080"),
	}
}

func (a Archive) HostURL() string { return a.hostURL }

func (a Archive) NetworkURL() string { return a.networkURL }

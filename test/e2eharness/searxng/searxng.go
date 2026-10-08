//go:build e2e

// Package searxng starts a disposable SearXNG container for e2e suites, configured
// with a caller-supplied settings.yml and any plugin or engine files it should mount.
package searxng

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerurl"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
)

const (
	Port              = "8080/tcp"
	settingsMountPath = "/etc/searxng/settings.yml"
	envImage          = "SEARXNG_IMAGE"
)

type Config struct {
	Alias        string
	SettingsYAML string
	Env          map[string]string
	Files        []testcontainers.ContainerFile
}

func Start(t *testing.T, ctx context.Context, networkName string, cfg Config) string {
	t.Helper()
	image := requiredimage.FromEnv(t, envImage, "searxng", "e2e")
	files := append([]testcontainers.ContainerFile{}, cfg.Files...)
	files = append(files, testcontainers.ContainerFile{
		Reader:            strings.NewReader(cfg.SettingsYAML),
		ContainerFilePath: settingsMountPath,
		FileMode:          0o644,
	})

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          image,
			ExposedPorts:   []string{Port},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {cfg.Alias}},
			Env:            cfg.Env,
			Files:          files,
			WaitingFor: wait.ForHTTP("/").
				WithPort(Port).
				WithStartupTimeout(2 * time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("start searxng container %s: %v", image, err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "searxng", container)
	return containerurl.HostURL(t, ctx, container, Port)
}

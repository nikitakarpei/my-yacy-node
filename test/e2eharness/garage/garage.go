//go:build e2e

// Package garage starts a disposable single-node Garage container that serves the S3 API
// for e2e suites, with one access key that may create buckets.
package garage

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
)

const (
	AccessKeyID     = "GK0123456789abcdef01234567"
	SecretAccessKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	image  = "docker.io/dxflrs/garage:v2.4.1"
	alias  = "garage"
	config = `metadata_dir = "/var/lib/garage/meta"
data_dir = "/var/lib/garage/data"
db_engine = "sqlite"
replication_factor = 1
rpc_bind_addr = "[::]:3901"
rpc_secret = "00000000000000000000000000000000000000000000000000000000000000e2"

[s3_api]
api_bind_addr = "[::]:3900"
s3_region = "us-east-1"
`
)

func Start(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          image,
			Cmd:            []string{"/garage", "server", "--single-node", "--default-access-key"},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {alias}},
			Env: map[string]string{
				"GARAGE_DEFAULT_ACCESS_KEY": AccessKeyID,
				"GARAGE_DEFAULT_SECRET_KEY": SecretAccessKey,
			},
			Files: []testcontainers.ContainerFile{{
				Reader:            strings.NewReader(config),
				ContainerFilePath: "/etc/garage.toml",
				FileMode:          0o644,
			}},
			WaitingFor: wait.ForLog("S3 API server listening").WithStartupTimeout(time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("start Garage container %s: %v", image, err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, alias, container)
}

func NetworkURL() string {
	return "http://" + alias + ":3900"
}

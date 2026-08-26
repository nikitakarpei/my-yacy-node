//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
)

const envWebArchiveScrapeImage = "WEBARCHIVESCRAPE_IMAGE"

func runWebArchiveScrape(
	t *testing.T,
	ctx context.Context,
	networkName string,
	arguments []string,
	env map[string]string,
) string {
	t.Helper()
	output := &commandOutput{}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image: requiredimage.FromEnv(
				t,
				envWebArchiveScrapeImage,
				"webarchivescrape",
				"e2e-webarchivescrape",
			),
			Cmd:      arguments,
			Env:      env,
			Networks: []string{networkName},
			LogConsumerCfg: &testcontainers.LogConsumerConfig{
				Consumers: []testcontainers.LogConsumer{output},
			},
			WaitingFor: wait.ForExit().WithExitTimeout(2 * time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("run webarchivescrape container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	state, err := container.State(ctx)
	if err != nil {
		t.Fatalf("read webarchivescrape exit state: %v", err)
	}
	if state.ExitCode != 0 {
		t.Fatalf("webarchivescrape exited %d: %s", state.ExitCode, output.stderrText())
	}
	return strings.TrimSpace(output.stdoutText())
}

type commandOutput struct {
	mu     sync.Mutex
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func (o *commandOutput) Accept(log testcontainers.Log) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if log.LogType == testcontainers.StdoutLog {
		_, _ = o.stdout.Write(log.Content)
		return
	}
	_, _ = o.stderr.Write(log.Content)
}

func (o *commandOutput) stdoutText() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.stdout.String()
}

func (o *commandOutput) stderrText() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.stderr.String()
}

func commandArguments(pywbURL, queriedURL string, dryRun bool) []string {
	arguments := []string{"-pywb-url", pywbURL, "-pywb-collection", "captures", "-url", queriedURL}
	if dryRun {
		arguments = append(arguments, "-dry-run")
	}
	return arguments
}

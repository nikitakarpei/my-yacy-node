package foregroundprocess_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/foregroundprocess"
)

func TestProcessReportsTheCommandExitStatus(t *testing.T) {
	process := started(t, []string{"/bin/sh", "-c", "exit 7"}, nil)

	if status := completedStatus(t, process); status != 7 {
		t.Errorf("ExitStatus = %d, want 7", status)
	}
}

func TestProcessRunsTheCommandWithTheGivenEnvironment(t *testing.T) {
	process := started(t,
		[]string{"/bin/sh", "-c", `test "$LEASED_NAME" = leased.example`},
		[]string{"LEASED_NAME=leased.example"},
	)

	if status := completedStatus(t, process); status != 0 {
		t.Errorf("ExitStatus = %d, want 0", status)
	}
}

func TestProcessDoesNotInheritTheParentEnvironment(t *testing.T) {
	t.Setenv("LEASED_NAME", "inherited.example")

	process := started(t, []string{"/bin/sh", "-c", `test -z "$LEASED_NAME"`}, nil)

	if status := completedStatus(t, process); status != 0 {
		t.Errorf("ExitStatus = %d, want 0", status)
	}
}

func TestProcessTerminatesTheCommandThatDoesNotExitOnItsOwn(t *testing.T) {
	process := started(t, []string{"sleep", "30"}, nil)

	process.Terminate(t.Context(), time.Second)

	if status := completedStatus(t, process); status == 0 {
		t.Error("ExitStatus = 0, want the status of a terminated command")
	}
}

func TestProcessRejectsACommandThatIsNotOnThePath(t *testing.T) {
	_, err := foregroundprocess.Start(
		t.Context(), []string{"no-such-command-in-this-image"}, nil,
	)
	if err == nil {
		t.Fatal("Start accepted a command that is not on the path")
	}
}

func started(
	t *testing.T,
	command []string,
	processEnvironment []string,
) *foregroundprocess.Process {
	t.Helper()

	process, err := foregroundprocess.Start(t.Context(), command, processEnvironment)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	return process
}

func completedStatus(t *testing.T, process *foregroundprocess.Process) int {
	t.Helper()

	select {
	case <-process.Completed():
		return process.ExitStatus()
	case <-time.After(30 * time.Second):
		process.Forward(context.Background(), os.Kill)
		t.Fatal("the command did not complete")

		return 0
	}
}

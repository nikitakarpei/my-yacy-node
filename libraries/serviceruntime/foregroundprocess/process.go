// Package foregroundprocess starts the one command a container exists to run
// and stands between it and the operating system. It forwards the signals the
// container receives to the command, terminates the command with a grace
// period, and reports the exit status the container must exit with.
package foregroundprocess

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	signaledExitStatusBase = 128
	waitFaultExitStatus    = 1
)

type Process struct {
	command    *exec.Cmd
	completed  chan struct{}
	exitStatus int
}

func Start(ctx context.Context, command []string, processEnvironment []string) (*Process, error) {
	//nolint:gosec // running the command the image is configured with is the whole job
	started := exec.CommandContext(ctx, command[0], command[1:]...)
	started.Env = append([]string{}, processEnvironment...)
	started.Stdin = os.Stdin
	started.Stdout = os.Stdout
	started.Stderr = os.Stderr

	if err := started.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", command[0], err)
	}

	process := &Process{command: started, completed: make(chan struct{})}
	go process.complete(context.WithoutCancel(ctx))

	slog.DebugContext(ctx, "foreground process started",
		slog.String("command", command[0]),
		slog.Int("pid", started.Process.Pid),
	)

	return process, nil
}

func (process *Process) complete(ctx context.Context) {
	err := process.command.Wait()
	process.exitStatus = exitStatusOf(process.command.ProcessState)

	if process.command.ProcessState == nil {
		slog.ErrorContext(ctx, "foreground process not waited for", slog.Any("error", err))
	}

	slog.DebugContext(ctx, "foreground process completed",
		slog.Int("exitStatus", process.exitStatus),
	)
	close(process.completed)
}

func exitStatusOf(state *os.ProcessState) int {
	if state == nil {
		return waitFaultExitStatus
	}

	if status, isWaitStatus := state.Sys().(syscall.WaitStatus); isWaitStatus &&
		status.Signaled() {
		return signaledExitStatusBase + int(status.Signal())
	}

	return state.ExitCode()
}

func (process *Process) Completed() <-chan struct{} {
	return process.completed
}

func (process *Process) ExitStatus() int {
	return process.exitStatus
}

func (process *Process) Forward(ctx context.Context, received os.Signal) {
	if err := process.command.Process.Signal(received); err != nil &&
		!errors.Is(err, os.ErrProcessDone) {
		slog.WarnContext(ctx, "signal not delivered to the foreground process",
			slog.String("signal", received.String()),
			slog.Any("error", err),
		)

		return
	}

	slog.DebugContext(ctx, "signal delivered to the foreground process",
		slog.String("signal", received.String()),
	)
}

func (process *Process) Terminate(ctx context.Context, grace time.Duration) {
	process.Forward(ctx, syscall.SIGTERM)

	timer := time.NewTimer(grace)
	defer timer.Stop()

	select {
	case <-process.completed:
		return
	case <-timer.C:
		slog.WarnContext(ctx, "foreground process did not exit within the grace period",
			slog.Duration("grace", grace),
		)
	}

	process.Forward(ctx, os.Kill)
	<-process.completed
}

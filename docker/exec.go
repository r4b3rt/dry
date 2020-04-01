package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
)

type ExecConfig struct {
	execConfig    types.ExecConfig
	Cmd           []string
	Height, Width uint
}

// Exec runs docker exec on a container.
func (daemon *Daemon) Exec(ctx context.Context, cid string, config ExecConfig) error {

	execConfig := types.ExecConfig{
		Tty:          true,
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		Cmd:          config.Cmd,
	}

	if _, err := daemon.client.ContainerInspect(ctx, cid); err != nil {
		return err
	}

	response, err := daemon.client.ContainerExecCreate(ctx, cid, execConfig)
	if err != nil {
		return err
	}

	if response.ID == "" {
		return errors.New("exec ID empty")
	}
	config.execConfig = execConfig

	return daemon.interactiveExec(ctx, config, response.ID)
}

func (daemon *Daemon) interactiveExec(ctx context.Context, config ExecConfig, execID string) error {
	var (
		out, stderr io.Writer
		in          io.ReadCloser
	)

	in = os.Stdin
	out = os.Stdout
	stderr = os.Stderr

	execStartCheck := types.ExecStartCheck{
		Tty: config.execConfig.Tty,
	}
	resp, err := daemon.client.ContainerExecAttach(ctx, execID, execStartCheck)
	if err != nil {
		return err
	}
	defer resp.Close()

	err = daemon.client.ContainerExecResize(ctx, execID, types.ResizeOptions{
		Height: config.Height,
		Width:  config.Width,
	})
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)
		errCh <- func() error {
			streamer := hijackedIOStreamer{
				inputStream:  in,
				outputStream: out,
				errorStream:  stderr,
				resp:         resp,
				tty:          config.execConfig.Tty,
			}

			return streamer.stream(ctx)
		}()
	}()

	if err := <-errCh; err != nil {
		return err
	}

	return daemon.getExecExitStatus(ctx, execID)
}

func (daemon *Daemon) getExecExitStatus(ctx context.Context, execID string) error {
	resp, err := daemon.client.ContainerExecInspect(ctx, execID)
	if err != nil {
		return err
	}
	if resp.ExitCode != 0 {
		return fmt.Errorf("unexpected exit code: %d", resp.ExitCode)
	}
	return nil
}

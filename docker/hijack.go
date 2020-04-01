package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
)

// A hijackedIOStreamer handles copying input to and output from streams to the
// connection.
type hijackedIOStreamer struct {
	inputStream  io.ReadCloser
	outputStream io.Writer
	errorStream  io.Writer

	resp types.HijackedResponse

	tty bool
}

// stream handles setting up the IO and then begins streaming stdin/stdout
// to/from the hijacked connection.
func (h *hijackedIOStreamer) stream(ctx context.Context) error {
	outputDone := h.beginOutputStream()
	inputDone := h.beginInputStream()

	select {
	case err := <-outputDone:
		return err
	case <-inputDone:
		// Input stream has closed.
		if h.outputStream != nil || h.errorStream != nil {
			select {
			case err := <-outputDone:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *hijackedIOStreamer) beginOutputStream() <-chan error {
	if h.outputStream == nil && h.errorStream == nil {
		return nil
	}

	outputDone := make(chan error)
	go func() {
		_, err := io.Copy(h.outputStream, h.resp.Reader)
		outputDone <- err
	}()

	return outputDone
}

func (h *hijackedIOStreamer) beginInputStream() <-chan error {
	inputDone := make(chan error)

	go func() {
		if h.inputStream != nil {
			if _, err := io.Copy(h.resp.Conn, h.inputStream); err != nil {
				inputDone <- fmt.Errorf("error sendStdin: %w", err)
			}
		}

		if err := h.resp.CloseWrite(); err != nil {
			inputDone <- fmt.Errorf("couldn't send EOF: %w", err)

		}

		close(inputDone)
	}()

	return inputDone
}

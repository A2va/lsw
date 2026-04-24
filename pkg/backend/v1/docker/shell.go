package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"charm.land/log/v2"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/moby/moby/client"
	"github.com/moby/term"

	"github.com/A2va/lsw/pkg/config"
)

func attachMethod(ctx context.Context, c *client.Client, nameOrID string) (client.HijackedResponse, error) {
	res, err := c.ContainerAttach(ctx, nameOrID, client.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
	})
	if err != nil {
		return client.HijackedResponse{}, err
	}

	// Prompt is not displayed correctly without it
	// res.HijackedResponse.Conn.Write([]byte("\r"))
	return res.HijackedResponse, nil
}

func execMethod(ctx context.Context, c *client.Client, nameOrID string, cmd string) error {

	command := []string{
		"wine",
		"cmd",
		"/C",
		cmd,
	}

	execConfig := client.ExecCreateOptions{
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
		// TTY:          true,
	}

	execIDResp, err := c.ExecCreate(ctx, nameOrID, execConfig)
	if err != nil {
		return err
	}

	attachResp, err := c.ExecAttach(ctx, execIDResp.ID, client.ExecAttachOptions{
		// TTY: true,
	})
	if err != nil {
		return err
	}

	defer attachResp.Close()

	// Demultiplex stdout and stderr
	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, attachResp.Reader)
	return nil
}

func Shell(bottle *config.Bottle, cmd string) error {
	log.Info("shelling into container", "name", bottle.Name)

	c, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	createOpts, err := createOptions(*bottle)
	if err != nil {
		return err
	}

	containerName := createOpts.Name

	_, err = c.ContainerCreate(ctx, createOpts)
	if err != nil {
		return err
	}

	_, err = c.ContainerStart(ctx, containerName, client.ContainerStartOptions{})
	if err != nil {
		return err
	}

	defer func() {
		// Use a fresh context for cleanup to ensure it runs even if a parent context was cancelled.
		cleanupCtx := context.Background()

		_, err = c.ContainerStop(cleanupCtx, containerName, client.ContainerStopOptions{})
		_, err = c.ContainerRemove(cleanupCtx, containerName, client.ContainerRemoveOptions{Force: true})
	}()

	// Non interactive
	if cmd != "" {
		errChan := make(chan error, 1)
		go func() {
			errChan <- execMethod(ctx, c, containerName, cmd)
		}()

		// Wait for command completion OR a system signal
		select {
		case err := <-errChan:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	res, err := attachMethod(ctx, c, containerName)
	if err != nil {
		return err
	}
	defer res.Close()

	fd := os.Stdin.Fd()
	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			return err
		}
		// Restore the terminal to normal when we exit
		defer term.RestoreTerminal(fd, oldState)
	}

	outputDone := make(chan error)

	go func() {
		_, err := io.Copy(os.Stdout, res.Reader)
		outputDone <- err
	}()

	go func() {
		_, err := io.Copy(res.Conn, os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stdin copy error: %v\n", err)
		}
	}()

	select {
	case err := <-outputDone:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

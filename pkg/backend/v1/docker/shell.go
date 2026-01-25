package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	"github.com/moby/moby/client"
	"github.com/moby/term"
)

func Shell(bottle config.Bottle) error {
	log.Debug("access to shell")
	c, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}

	containerID, err := getContainerID(c, bottle.Name)
	if err != nil {
		return err
	}

	execConfig := client.ExecCreateOptions{
		Cmd:          []string{"wine", "cmd"},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
	}

	execIDResp, err := c.ExecCreate(context.Background(), containerID, execConfig)
	if err != nil {
		panic(fmt.Errorf("failed to create exec: %v", err))
	}

	attachResp, err := c.ExecAttach(context.Background(), execIDResp.ID, client.ExecAttachOptions{
		TTY: true,
	})
	if err != nil {
		panic(fmt.Errorf("failed to attach to exec: %v", err))
	}
	defer attachResp.Close()

	// Set up the local terminal for interactive use (Raw Mode)
	// This ensures that Ctrl+C, arrows, etc., are passed to the container
	// instead of handled by your local shell.
	fd := os.Stdin.Fd()
	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			panic(err)
		}
		// Restore the terminal to normal when we exit
		defer term.RestoreTerminal(fd, oldState)
	}

	// Stream Input/Output
	// We use a channel to know when the remote command has finished.
	outputDone := make(chan error)

	// Copy container output -> local stdout
	go func() {
		_, err := io.Copy(os.Stdout, attachResp.Reader)
		outputDone <- err
	}()

	// Copy local stdin -> container input
	go func() {
		_, err := io.Copy(attachResp.Conn, os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stdin copy error: %v\n", err)
		}
	}()

	// Wait for the output stream to finish (which means the command exited)
	<-outputDone

	return nil
}

package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"charm.land/log/v2"
	"github.com/A2va/lsw/pkg/config"
	"github.com/moby/moby/client"
	"github.com/moby/term"
)

func attachMethod(c *client.Client, nameOrID string) (client.HijackedResponse, error) {
	res, err := c.ContainerAttach(context.Background(), nameOrID, client.ContainerAttachOptions{
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

func execMethod(c *client.Client, nameOrID string) (client.HijackedResponse, error) {
	execConfig := client.ExecCreateOptions{
		Cmd:          []string{"wine", "cmd"},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
	}

	execIDResp, err := c.ExecCreate(context.Background(), nameOrID, execConfig)
	if err != nil {
		return client.HijackedResponse{}, err
	}

	attachResp, err := c.ExecAttach(context.Background(), execIDResp.ID, client.ExecAttachOptions{
		TTY: true,
	})
	if err != nil {
		return client.HijackedResponse{}, err
	}
	return attachResp.HijackedResponse, nil
}

func Shell(bottle *config.Bottle) error {
	// FIXME Cannot create two shell session of the same bottle
	log.Info("shelling into container (docker)", "name", bottle.Name)

	c, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}

	createOpts, err := createOptions(*bottle)
	if err != nil {
		return err
	}

	containerName := createOpts.Name

	_, err = c.ContainerCreate(context.Background(), createOpts)
	if err != nil {
		return err
	}

	res, err := attachMethod(c, containerName)
	if err != nil {
		return err
	}
	defer res.Close()

	_, err = c.ContainerStart(context.Background(), containerName, client.ContainerStartOptions{})
	if err != nil {
		return err
	}

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

	<-outputDone

	_, err = c.ContainerStop(context.Background(), containerName, client.ContainerStopOptions{})
	if err != nil {
		return err
	}

	_, err = c.ContainerRemove(context.Background(), containerName, client.ContainerRemoveOptions{})
	if err != nil {
		return err
	}

	return nil
}

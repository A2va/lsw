package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/A2va/lsw/pkg/config"
	"github.com/moby/moby/client"
	"github.com/moby/term"
)

func attachMethod(c *client.Client, containerID string) (client.HijackedResponse, error) {
	res, err := c.ContainerAttach(context.Background(), containerID, client.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
	})
	if err != nil {
		return client.HijackedResponse{}, err
	}

	res.HijackedResponse.Conn.Write([]byte("\r"))
	return res.HijackedResponse, nil
}

func execMethod(c *client.Client, containerID string) (client.HijackedResponse, error) {
	execConfig := client.ExecCreateOptions{
		Cmd:          []string{"wine", "cmd"},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
	}

	execIDResp, err := c.ExecCreate(context.Background(), containerID, execConfig)
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

func Shell(bottle config.Bottle) error {
	c, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}

	containerID, err := getContainerID(c, bottle.Name)
	if err != nil {
		return err
	}

	res, err := execMethod(c, containerID)
	// res, err := attachMethod(c, containerID)
	if err != nil {
		return err
	}
	defer res.Close()

	fd := os.Stdin.Fd()
	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			panic(err)
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

	return nil
}

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

	stream, err := c.ContainerAttach(context.Background(), containerID, client.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
	})
	if err != nil {
		return err
	}

	defer stream.Close()

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
		_, err := io.Copy(os.Stdout, stream.Reader)
		outputDone <- err
	}()

	go func() {
		_, err := io.Copy(stream.Conn, os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stdin copy error: %v\n", err)
		}
	}()

	<-outputDone

	return nil
}

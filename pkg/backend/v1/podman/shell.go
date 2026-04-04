package podman

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"charm.land/log/v2"
	"github.com/containers/podman/v6/pkg/api/handlers"
	"github.com/containers/podman/v6/pkg/bindings/containers"
	"github.com/docker/docker/api/types/container"

	"github.com/A2va/lsw/pkg/config"
)

func attachMethod(c context.Context, nameOrID string, attachReady chan bool) error {
	err := containers.Attach(c, nameOrID, os.Stdin, os.Stdout, os.Stderr, attachReady, &containers.AttachOptions{})
	if err != nil {
		return err
	}
	return nil
}

func execMethod(c context.Context, nameOrID string, cmd string) error {

	command := []string{
		"wine",
		"cmd",
		"/C",
		cmd,
	}

	execConfig := handlers.ExecCreateConfig{
		ExecOptions: container.ExecOptions{
			Cmd:          command,
			AttachStdout: true,
			AttachStderr: true,
			// Tty:          true,
		},
	}

	execID, err := containers.ExecCreate(c, nameOrID, &execConfig)
	if err != nil {
		return err
	}

	opts := new(containers.ExecStartAndAttachOptions)
	opts.WithInputStream(*bufio.NewReader(os.Stdin))
	opts.WithOutputStream(os.Stdout)
	opts.WithAttachInput(true)
	opts.WithAttachOutput(true)
	opts.WithAttachError(true)

	if err := containers.ExecStartAndAttach(c, execID, opts); err != nil {
		return fmt.Errorf("attach failed: %w", err)
	}
	return nil
}

func Shell(bottle *config.Bottle, cmd string) error {
	log.Info("shelling into container (podman)", "name", bottle.Name)

	c, err := podmanClient()
	if err != nil {
		return err
	}

	spec, err := createSpec(*bottle)
	if err != nil {
		return err
	}

	containerName := spec.Name

	_, err = containers.CreateWithSpec(c, &spec, &containers.CreateOptions{})
	if err != nil {
		return err
	}

	err = containers.Start(c, containerName, &containers.StartOptions{})
	if err != nil {
		return err
	}

	// Non interactive
	if cmd != "" {
		return execMethod(c, containerName, cmd)
	}

	attachReady := make(chan bool, 1)
	attachErr := make(chan error, 1)

	// Hook up the streams in the background
	go func() {
		attachErr <- attachMethod(c, containerName, attachReady)
	}()

	// Wait until Podman signals that it is actively listening
	<-attachReady

	return <-attachErr
}

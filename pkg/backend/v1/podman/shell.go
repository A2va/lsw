package podman

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	"github.com/containers/podman/v6/pkg/api/handlers"
	"github.com/containers/podman/v6/pkg/bindings"
	"github.com/containers/podman/v6/pkg/bindings/containers"
	"github.com/docker/docker/api/types/container"
)

func attachMethod(c context.Context, nameOrID string) error {
	attachReady := make(chan bool)
	err := containers.Attach(c, nameOrID, os.Stdin, os.Stdout, os.Stderr, attachReady, &containers.AttachOptions{})
	if err != nil {
		return err
	}
	return nil
}

func execMethod(c context.Context, nameOrID string) error {
	execConfig := handlers.ExecCreateConfig{
		ExecOptions: container.ExecOptions{
			Cmd:          []string{"wine", "cmd"},
			AttachStdin:  true,
			AttachStdout: true,
			Tty:          true,
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

	if err := containers.ExecStartAndAttach(c, execID, opts); err != nil {
		return fmt.Errorf("attach failed: %w", err)
	}
	return nil
}

func Shell(bottle *config.Bottle) error {
	// FIXME Cannot create two shell session of the same bottle
	log.Info("shelling into container (podman)", "name", bottle.Name)

	c, err := bindings.NewConnection(context.Background(), "unix:///run/podman/podman.sock")
	if err != nil {
		return err
	}

	spec, err := CreateSpec(*bottle)
	if err != nil {
		return err
	}

	_, err = containers.CreateWithSpec(c, &spec, &containers.CreateOptions{})
	if err != nil {
		return err
	}

	err = containers.Start(c, bottle.Name, &containers.StartOptions{})
	if err != nil {
		return err
	}

	err = attachMethod(c, bottle.Name)
	if err != nil {
		return err
	}

	err = containers.Stop(c, bottle.Name, &containers.StopOptions{})
	if err != nil {
		return err
	}

	return nil
}

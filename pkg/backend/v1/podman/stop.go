package podman

import (
	"context"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	"github.com/containers/podman/v6/pkg/bindings"
	"github.com/containers/podman/v6/pkg/bindings/containers"
)

func Stop(bottle config.Bottle) error {
	log.Debug("stop container using podman provider", "name", bottle.Name)

	c, err := bindings.NewConnection(context.Background(), "unix:///run/podman/podman.sock")
	if err != nil {
		return err
	}

	var t uint = 1
	err = containers.Stop(c, bottle.Name, &containers.StopOptions{Timeout: &t})
	if err != nil {
		return err
	}

	return nil
}

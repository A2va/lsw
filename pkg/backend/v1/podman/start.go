package podman

import (
	"context"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	"github.com/containers/podman/v6/pkg/bindings"
	"github.com/containers/podman/v6/pkg/bindings/containers"
)

func Start(bottle config.Bottle) error {
	log.Debug("start container using podman provider", "name", bottle.Name)

	c, err := bindings.NewConnection(context.Background(), "unix:///run/podman/podman.sock")
	if err != nil {
		return err
	}

	err = containers.Start(c, bottle.Name, &containers.StartOptions{})
	if err != nil {
		return err
	}

	return nil
}

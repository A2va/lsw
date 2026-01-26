package podman

import (
	"context"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	"github.com/containers/podman/v6/pkg/bindings"
	"github.com/containers/podman/v6/pkg/bindings/containers"
)

func getContainerID(c context.Context, name string) (string, error) {
	log.Debug("get container from", "name", name)

	f := map[string][]string{"reference": []string{name}}
	t := true
	res, err := containers.List(c, &containers.ListOptions{All: &t, Filters: f})
	if err != nil {
		return "", err
	}
	return res[0].ID, nil
}

func Start(bottle config.Bottle) error {
	log.Debug("start container using podman provider", "name", bottle.Name)

	c, err := bindings.NewConnection(context.Background(), "unix:///run/podman/podman.sock")
	if err != nil {
		return err
	}

	// TODO Remove getContainerID
	containerID, err := getContainerID(c, bottle.Name)
	if err != nil {
		return err
	}

	err = containers.Start(c, containerID, &containers.StartOptions{})
	if err != nil {
		return err
	}

	return nil
}

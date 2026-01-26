package docker

import (
	"context"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	"github.com/moby/moby/client"
)

func Start(bottle config.Bottle) error {
	log.Debug("start container using docker provider", "name", bottle.Name)

	c, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}

	log.Debug("start v1 bottle", "name", bottle.Name)
	_, err = c.ContainerStart(context.Background(), bottle.Name, client.ContainerStartOptions{})
	if err != nil {
		return err
	}
	return nil
}

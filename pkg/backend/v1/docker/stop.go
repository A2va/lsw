package docker

import (
	"context"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	"github.com/moby/moby/client"
)

func Stop(bottle config.Bottle) error {
	log.Debug("stop container using docker provider", "name", bottle.Name)

	c, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}

	log.Debug("start v1 bottle", "name", bottle.Name)
	t := 1
	_, err = c.ContainerStop(context.Background(), bottle.Name, client.ContainerStopOptions{Timeout: &t})
	if err != nil {
		return err
	}
	return nil
}

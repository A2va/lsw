package docker

import (
	"context"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	"github.com/moby/moby/client"
)

func getContainerID(c *client.Client, name string) (string, error) {
	log.Debug("get container from", "name", name)

	f := make(client.Filters).Add("name", name)
	res, err := c.ContainerList(context.Background(), client.ContainerListOptions{All: true, Filters: f})

	if err != nil {
		return "", err
	}

	return res.Items[0].ID, nil
}

func Start(bottle config.Bottle) error {
	log.Debug("start container using docker provider", "name", bottle.Name)

	c, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}

	containerID, err := getContainerID(c, bottle.Name)
	if err != nil {
		return err
	}

	log.Debug("start v1 bottle", "name", bottle.Name)
	_, err = c.ContainerStart(context.Background(), containerID, client.ContainerStartOptions{})
	if err != nil {
		return err
	}
	return nil
}

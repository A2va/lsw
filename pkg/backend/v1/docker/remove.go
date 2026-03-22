package docker

import (
	"context"
	"fmt"

	"github.com/A2va/lsw/pkg/config"
	"github.com/moby/moby/client"
)

func Remove(bottle *config.Bottle) error {
	c, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}

	volumeName := fmt.Sprintf("lsw-v1-%s", bottle.Name)
	_, err = c.VolumeRemove(context.Background(), volumeName, client.VolumeRemoveOptions{})
	if err != nil {
		return err
	}

	config.Get().RemoveBottle(bottle.Name)
	return nil
}

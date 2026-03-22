package podman

import (
	"fmt"

	"github.com/A2va/lsw/pkg/config"
	"github.com/containers/podman/v6/pkg/bindings/volumes"
)

func Remove(bottle *config.Bottle) error {
	c, err := podmanClient()
	if err != nil {
		return err
	}

	volumeName := fmt.Sprintf("lsw-v1-%s", bottle.Name)
	err = volumes.Remove(c, volumeName, &volumes.RemoveOptions{})
	if err != nil {
		return err
	}
	config.Get().RemoveBottle(bottle.Name)
	return nil
}

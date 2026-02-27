package podman

import (
	"fmt"

	"github.com/A2va/lsw/pkg/config"
	"github.com/containers/podman/v6/pkg/bindings/volumes"
)

func Remove(bottle *config.Bottle) error {
	// FIXME remove bottle
	c, err := podmanClient()
	if err != nil {
		return err
	}

	volumeName := fmt.Sprintf("lsw-v1-%s", bottle.Name)
	return volumes.Remove(c, volumeName, &volumes.RemoveOptions{})
}

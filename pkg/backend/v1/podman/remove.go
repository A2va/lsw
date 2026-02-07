package podman

import (
	"context"
	"fmt"

	"github.com/A2va/lsw/pkg/config"
	"github.com/containers/podman/v6/pkg/bindings"
	"github.com/containers/podman/v6/pkg/bindings/volumes"
)

func Remove(bottle config.Bottle) error {
	c, err := bindings.NewConnection(context.Background(), "unix:///run/podman/podman.sock")
	if err != nil {
		return err
	}

	volumeName := fmt.Sprintf("lsw-v1-%s", bottle.Name)
	return volumes.Remove(c, volumeName, &volumes.RemoveOptions{})
}

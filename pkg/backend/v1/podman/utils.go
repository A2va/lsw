package podman

import (
	"context"
	"fmt"
	"os"

	"charm.land/log/v2"
	"github.com/containers/podman/v6/pkg/bindings"
)

func podmanClient() (context.Context, error) {
	uri := fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", os.Geteuid())
	log.Debug("podman socket", "uri", uri)
	c, err := bindings.NewConnection(context.Background(), uri)
	if err != nil {
		return nil, err
	}
	return c, nil

}

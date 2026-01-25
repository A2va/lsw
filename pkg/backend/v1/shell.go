package v1

import (
	"github.com/A2va/lsw/pkg/backend/v1/docker"
	"github.com/A2va/lsw/pkg/config"
)

func Shell(bottle config.Bottle) error {
	if bottle.V1Provider == "docker" {
		return docker.Shell(bottle)
	}
	return nil
}

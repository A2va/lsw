package v1

import (
	"github.com/A2va/lsw/pkg/backend/v1/docker"
	"github.com/A2va/lsw/pkg/backend/v1/podman"
	"github.com/A2va/lsw/pkg/config"
)

func Stop(bottle config.Bottle) error {
	if bottle.V1Provider == "docker" {
		return docker.Stop(bottle)
	} else if bottle.V1Provider == "podman" {
		return podman.Stop(bottle)
	}
	return nil
}

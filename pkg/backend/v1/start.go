package v1

import (
	"github.com/A2va/lsw/pkg/backend/v1/docker"
	"github.com/A2va/lsw/pkg/backend/v1/podman"
	"github.com/A2va/lsw/pkg/config"
)

func Start(bottle *config.Bottle) error {
	if bottle.V1Provider == "docker" {
		return docker.Start(bottle)
	} else if bottle.V1Provider == "podman" {
		return podman.Start(bottle)
	}
	return nil
}

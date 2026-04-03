package v1

import (
	"github.com/A2va/lsw/pkg/backend/v1/docker"
	"github.com/A2va/lsw/pkg/backend/v1/podman"
	"github.com/A2va/lsw/pkg/config"
)

func Shell(bottle *config.Bottle, cmd string) error {
	if bottle.V1Provider == "docker" {
		return docker.Shell(bottle, cmd)
	} else if bottle.V1Provider == "podman" {
		return podman.Shell(bottle, cmd)
	}
	return nil
}

package v1

import (
	"github.com/A2va/lsw/pkg/backend/v1/docker"
	"github.com/A2va/lsw/pkg/backend/v1/podman"
	"github.com/A2va/lsw/pkg/config"
)

func Init() {
	// no need to send a specific bottle since init for bottle creation
	provider := getProvider(config.Bottle{})

	if provider == "docker" {
		docker.Init()
	} else if provider == "podman" {
		podman.Init()
	}
}

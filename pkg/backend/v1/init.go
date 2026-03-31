package v1

import (
	"github.com/A2va/lsw/pkg/backend/v1/docker"
	"github.com/A2va/lsw/pkg/backend/v1/podman"
	"github.com/A2va/lsw/pkg/config"
	"github.com/A2va/lsw/pkg/utils"
)

func Init(provider string) {
	// no need to send a specific bottle since init for bottle creation
	if provider == "" {
		provider = getProvider(config.Bottle{})
	}

	// TODO Move this inside provider implementation
	progressCallback := utils.GetProgressCallback()
	if progressCallback != nil {
		progressCallback("Building image", utils.ProgressStart)
	}

	if provider == "docker" {
		docker.Init()
	} else if provider == "podman" {
		podman.Init()
	}

	if progressCallback != nil {
		progressCallback("Build complete", utils.ProgressDone)
	}
}

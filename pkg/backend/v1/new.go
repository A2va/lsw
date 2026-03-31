package v1

import (
	"charm.land/log/v2"
	"github.com/A2va/lsw/pkg/backend/v1/docker"
	"github.com/A2va/lsw/pkg/backend/v1/podman"
	"github.com/A2va/lsw/pkg/config"
	"github.com/A2va/lsw/pkg/utils"
)

func New(name string, provider string) error {
	if provider == "" {
		provider = getProvider(config.Bottle{})
	}
	Init(provider)

	log.Info("creating new bottle (v1 backend)", "name", name)

	progressCallback := utils.GetProgressCallback()
	if progressCallback != nil {
		progressCallback("Creating bottle", utils.ProgressStart)
	}

	var err error
	if provider == "docker" {
		err = docker.New(name)
	} else if provider == "podman" {
		err = podman.New(name)
	}

	if err != nil {
		return err
	}

	log.Info("updating config to add new bottle")

	// Update the config
	config.Get().AddBottle(config.Bottle{
		Name:       name,
		Version:    "v1",
		V1Provider: provider,
	})

	if progressCallback != nil {
		progressCallback("Bottle created successfully", utils.ProgressDone)
	}
	return nil
}

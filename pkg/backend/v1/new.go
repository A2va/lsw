package v1

import (
	"github.com/A2va/lsw/pkg/backend/v1/docker"
	"github.com/A2va/lsw/pkg/backend/v1/podman"
	"github.com/A2va/lsw/pkg/config"
)

func New(name string) error {
	provider := getProvider(config.Bottle{})

	var err error
	if provider == "docker" {
		err = docker.New(name)
	} else if provider == "podman" {
		err = podman.New(name)
	}

	if err != nil {
		return err
	}

	cfg := config.Get()

	// Update the config
	cfg.Bottles = append(cfg.Bottles, config.Bottle{
		Name:       name,
		Version:    "v1",
		V1Provider: provider,
	})

	// Set the default bottle if not already set
	if cfg.DefaultBottle == "" {
		cfg.DefaultBottle = name
	}
	return nil
}

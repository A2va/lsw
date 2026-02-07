package v1

import (
	"github.com/A2va/lsw/pkg/backend/v1/docker"
	"github.com/A2va/lsw/pkg/backend/v1/podman"
	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
)

func New(name string, provider string) error {
	if provider == "" {
		provider = getProvider(config.Bottle{})
	}
	Init(provider)

	log.Info("creating new bottle (v1 backend)", "name", name)

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

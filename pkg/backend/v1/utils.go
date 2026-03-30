package v1

import (
	"fmt"
	"os/exec"

	"github.com/A2va/lsw/pkg/backend/v1/docker"
	"github.com/A2va/lsw/pkg/backend/v1/podman"
	"github.com/A2va/lsw/pkg/config"
)

// Find the first available provider
func findFirstAvailableProvider() string {
	binaries := []string{"docker", "podman"}
	var selectedBin string

	// Find the first available binary in the system PATH
	for _, bin := range binaries {
		if _, err := exec.LookPath(bin); err == nil {
			selectedBin = bin
			break
		}
	}
	return selectedBin
}

func getProvider(bottle config.Bottle) string {
	if bottle.V1Provider != "" {
		return bottle.V1Provider
	}

	cfg := config.Get()
	if cfg.DefaultV1Provider == "" {
		provider := findFirstAvailableProvider()
		cfg.DefaultV1Provider = provider
		return provider
	}
	return cfg.DefaultV1Provider
}

func GetStatus(bottle config.Bottle, all bool) ([]config.BottleStatus, error) {
	if bottle.V1Provider == "podman" {
		return podman.GetStatus(bottle.Name, all)
	} else if bottle.V1Provider == "docker" {
		return docker.GetStatus(bottle.Name, all)
	}

	return []config.BottleStatus{}, fmt.Errorf("not a valid v1 provider")
}

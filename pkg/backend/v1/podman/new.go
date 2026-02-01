package podman

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/A2va/lsw/pkg/config"
	"github.com/containers/podman/v6/libpod/define"
	"github.com/containers/podman/v6/pkg/bindings"
	"github.com/containers/podman/v6/pkg/bindings/containers"
	"github.com/containers/podman/v6/pkg/specgen"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func CreateSpec(bottle config.Bottle) (specgen.SpecGenerator, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return specgen.SpecGenerator{}, err
	}

	version := config.GetVersion()
	image := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)
	volumeName := fmt.Sprintf("lsw-v1-%s", bottle.Name)

	var mounts []specs.Mount

	mounts = append(mounts, specs.Mount{
		Type:        "volume",
		Source:      volumeName,
		Destination: "/opt/prefix",
	})

	mounts = append(mounts, specs.Mount{
		Type:        "bind",
		Source:      cwd,
		Destination: cwd,
	})

	for _, m := range bottle.Mounts {
		mountPath, err := filepath.Abs(m)
		if err != nil {
			return specgen.SpecGenerator{}, err
		}
		mounts = append(mounts, specs.Mount{Type: "bind", Source: mountPath, Destination: mountPath})
	}

	t := true
	spec := specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name:     bottle.Name,
			Command:  []string{"wine", "cmd"},
			Stdin:    &t,
			Terminal: &t,
		},
		ContainerStorageConfig: specgen.ContainerStorageConfig{
			Image:  image,
			Mounts: mounts,
		},
		ContainerHealthCheckConfig: specgen.ContainerHealthCheckConfig{
			HealthLogDestination: define.DefaultHealthCheckLocalDestination,
			HealthMaxLogCount:    define.DefaultHealthMaxLogCount,
			HealthMaxLogSize:     define.DefaultHealthMaxLogSize,
		},
	}

	return spec, nil
}

func New(name string) error {
	c, err := bindings.NewConnection(context.Background(), "unix:///run/podman/podman.sock")
	if err != nil {
		return err
	}

	spec, err := CreateSpec(config.Bottle{Name: name})
	if err != nil {
		return err
	}

	res, err := containers.CreateWithSpec(c, &spec, &containers.CreateOptions{})
	if err != nil {
		return err
	}

	// Remove the container because all we wanted was to create a new volume
	// From there container creation are faster, so that we can mount new path easily be recreating the container (they are immutable)
	_, err = containers.Remove(c, res.ID, &containers.RemoveOptions{})
	if err != nil {
		return err
	}

	return nil
}

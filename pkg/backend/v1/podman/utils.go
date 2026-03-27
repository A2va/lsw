package podman

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"charm.land/log/v2"
	"github.com/A2va/lsw/pkg/backend"
	"github.com/A2va/lsw/pkg/config"
	"github.com/containers/podman/v6/libpod/define"
	"github.com/containers/podman/v6/pkg/bindings"
	"github.com/containers/podman/v6/pkg/specgen"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func podmanClient() (context.Context, error) {
	uri := fmt.Sprintf("unix:///run/user/%d/podman/podman.sock", os.Geteuid())
	log.Debug("podman socket", "uri", uri)
	c, err := bindings.NewConnection(context.Background(), uri)
	if err != nil {
		return nil, err
	}
	return c, nil

}

func createSpec(bottle config.Bottle) (specgen.SpecGenerator, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return specgen.SpecGenerator{}, err
	}

	version := config.GetVersion()
	image := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)
	volumeName := fmt.Sprintf("lsw-v1-%s", bottle.Name)

	var mounts []specs.Mount
	var namedVolumes []*specgen.NamedVolume

	namedVolumes = append(namedVolumes, &specgen.NamedVolume{
		Name: volumeName,
		Dest: "/opt/prefix",
	})

	mounts = append(mounts, specs.Mount{
		Type:        "bind",
		Source:      cwd,
		Destination: cwd,
		Options:     []string{"rbind", "z"},
	})

	for _, m := range bottle.Mounts {
		mountPath, err := filepath.Abs(m)
		if err != nil {
			return specgen.SpecGenerator{}, err
		}
		mounts = append(mounts, specs.Mount{Type: "bind", Source: mountPath, Destination: mountPath, Options: []string{"rbind", "z"}})
	}

	t := true
	spec := specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name:     bottle.Name,
			Command:  []string{"wine", backend.GetShell(bottle)},
			Stdin:    &t,
			Terminal: &t,
		},
		ContainerStorageConfig: specgen.ContainerStorageConfig{
			Image:   image,
			Mounts:  mounts,
			Volumes: namedVolumes,
			WorkDir: cwd,
		},
		ContainerHealthCheckConfig: specgen.ContainerHealthCheckConfig{
			HealthLogDestination: define.DefaultHealthCheckLocalDestination,
			HealthMaxLogCount:    define.DefaultHealthMaxLogCount,
			HealthMaxLogSize:     define.DefaultHealthMaxLogSize,
		},
	}

	return spec, nil
}

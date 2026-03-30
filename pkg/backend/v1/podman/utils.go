package podman

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/A2va/lsw/pkg/config"
	"github.com/containers/podman/v6/libpod/define"
	"github.com/containers/podman/v6/pkg/bindings"
	"github.com/containers/podman/v6/pkg/bindings/containers"
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

func hash() string {
	t := time.Now().String()
	bs := sha1.Sum([]byte(t))
	return hex.EncodeToString(bs[:])[:7]
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

	name := fmt.Sprintf("lsw-%s-%s", bottle.Name, hash())
	log.Debug(name)

	t := true
	spec := specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name:     name,
			Command:  []string{"wine", bottle.GetShell()},
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

func GetStatus(name string, all bool) ([]config.BottleStatus, error) {
	a := true

	c, err := podmanClient()
	if err != nil {
		return []config.BottleStatus{}, err
	}

	f := map[string][]string{
		"ancestor": []string{"lsw-v1:.*"},                  // Matches "lsw-v1:<anything>"
		"name":     []string{fmt.Sprintf("^lsw-%s", name)}, // Matches names starting with "lsw-name"
	}

	containerss, err := containers.List(c, &containers.ListOptions{
		All:     &a,
		Filters: f,
	})
	if err != nil {
		return []config.BottleStatus{}, err
	}

	if len(containerss) == 0 && all {
		notRunning := config.BottleStatus{
			Name:    name,
			Running: false,
		}
		return []config.BottleStatus{notRunning}, nil
	}

	var status []config.BottleStatus
	for _, container := range containerss {
		inspect, err := containers.Inspect(c, container.ID, &containers.InspectOptions{})
		if err != nil {
			return []config.BottleStatus{}, err
		}

		status = append(status, config.BottleStatus{
			Name:        strings.TrimPrefix(inspect.Name, "lsw-"),
			Running:     true,
			EnteredFrom: inspect.Config.WorkingDir,
		})
	}

	return status, nil
}

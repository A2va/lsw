package podman

import (
	"context"
	"fmt"
	"os"

	"github.com/A2va/lsw/pkg/config"
	"github.com/containers/podman/v6/pkg/bindings"
	"github.com/containers/podman/v6/pkg/bindings/containers"
	"github.com/containers/podman/v6/pkg/specgen"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func New(name string) error {
	c, err := bindings.NewConnection(context.Background(), "unix:///run/podman/podman.sock")
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	version := config.GetVersion()
	image := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)
	s := specgen.NewSpecGenerator(image, false)

	s.Name = name
	s.WorkDir = "/mnt/workdir"
	s.Command = []string{"wine", "cmd.exe"}

	t := true
	s.Terminal = &t
	s.Stdin = &t
	s.Terminal = &t

	volumeName := fmt.Sprintf("lsw-v1-%s", name)
	s.Volumes = []*specgen.NamedVolume{
		{
			Name: volumeName,
			Dest: "/opt/prefix",
		},
	}

	s.Mounts = []specs.Mount{
		{
			Source:      cwd,
			Destination: "/mnt/workdir",
			Type:        "bind",
			// Options: []string{"rw", "z"}, // "z" is useful for SELinux
		},
	}

	res, err := containers.CreateWithSpec(c, s, nil)
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

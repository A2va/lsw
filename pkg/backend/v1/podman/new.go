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

	s.Volumes = []*specgen.NamedVolume{
		{
			Name: "lsw_wine_prefix",
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

	_, err = containers.CreateWithSpec(c, s, nil)
	if err != nil {
		return err
	}

	return nil
}

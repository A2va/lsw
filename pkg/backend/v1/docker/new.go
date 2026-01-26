package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

func New(name string) error {
	c, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	version := config.GetVersion()
	image := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)

	createOpts := client.ContainerCreateOptions{
		Name: name,
		Config: &container.Config{
			Image:     image,
			Cmd:       []string{"wine", "cmd"},
			Tty:       true,
			OpenStdin: true,
		},
		HostConfig: &container.HostConfig{
			Binds: []string{
				"lsw_wine_prefix:/opt/prefix",
				fmt.Sprintf("%s:/mnt/workdir", filepath.ToSlash(cwd)),
			},
		},
	}

	_, err = c.ContainerCreate(context.Background(), createOpts)
	if err != nil {
		return err
	}

	return nil
}

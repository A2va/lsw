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

	volumeName := fmt.Sprintf("lsw-v1-%s", name)
	bindName := fmt.Sprintf("%s:/opt/prefix", volumeName)

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
				bindName,
				fmt.Sprintf("%s:/mnt/workdir", filepath.ToSlash(cwd)),
			},
		},
	}

	res, err := c.ContainerCreate(context.Background(), createOpts)
	if err != nil {
		return err
	}

	// Remove the container because all we wanted was to create a new volume
	// From there container creation are faster, so that we can mount new path easily be recreating the container (they are immutable)
	_, err = c.ContainerRemove(context.Background(), res.ID, client.ContainerRemoveOptions{})
	if err != nil {
		return err
	}

	return nil
}

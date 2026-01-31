package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
)

func CreateOptions(bottle config.Bottle) (client.ContainerCreateOptions, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return client.ContainerCreateOptions{}, err
	}

	version := config.GetVersion()
	image := fmt.Sprintf("lsw-v1:%s", version.ShortCommit)
	volumeName := fmt.Sprintf("lsw-v1-%s", bottle.Name)

	var mounts []mount.Mount
	mounts = append(mounts, mount.Mount{
		Type:   mount.TypeVolume,
		Source: volumeName,
		Target: "/opt/prefix",
	})

	mounts = append(mounts, mount.Mount{
		Type:   mount.TypeBind,
		Source: cwd,
		Target: cwd,
	})

	for _, m := range bottle.Mounts {
		mountPath, err := filepath.Abs(m)
		if err != nil {
			return client.ContainerCreateOptions{}, err
		}
		mounts = append(mounts, mount.Mount{Type: mount.TypeBind, Source: mountPath, Target: mountPath})
	}

	createOpts := client.ContainerCreateOptions{
		Name: bottle.Name,
		Config: &container.Config{
			Image: image,
			Cmd:   []string{"wine", "cmd"},
			// Cmd:          []string{"bash"},
			Tty:          true,
			OpenStdin:    true,
			AttachStdin:  true,
			AttachStdout: true,
			WorkingDir:   cwd,
		},
		HostConfig: &container.HostConfig{
			Mounts: mounts,
		},
	}

	log.Debug("docker provider", "createOptions", createOpts)
	return createOpts, nil
}

func New(name string) error {
	c, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}

	createOpts, err := CreateOptions(config.Bottle{Name: name})
	if err != nil {
		return err
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

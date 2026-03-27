package docker

import (
	"context"
	"fmt"
	"os"

	"github.com/A2va/lsw/pkg/config"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

func New(name string) error {
	c, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}

	createOpts, err := createOptions(config.Bottle{Name: name})
	if err != nil {
		return err
	}

	// We must override the config to run as root specifically for this setup step
	// so we can change the ownership of the new volume.
	createOpts.Config.User = "0" // Run as root

	// Determine the host user's UID/GID to transfer ownership to.
	uid := os.Getuid()
	gid := os.Getgid()

	// Override the command to change ownership of the volume target.
	// We set Entrypoint to /bin/sh to ensure we ignore any Wine-specific entrypoints in the image.
	createOpts.Config.Entrypoint = []string{"/bin/sh", "-c"}
	createOpts.Config.Cmd = []string{fmt.Sprintf("chown -R %d:%d /opt/prefix", uid, gid)}

	// Disable interactive flags for this background maintenance task
	createOpts.Config.Tty = false
	createOpts.Config.OpenStdin = false
	createOpts.Config.AttachStdin = false

	res, err := c.ContainerCreate(context.Background(), createOpts)
	if err != nil {
		return err
	}

	// Ensure we remove the container even if the start fails
	defer func() {
		c.ContainerRemove(context.Background(), res.ID, client.ContainerRemoveOptions{})
	}()

	// Start the container to execute the chown command
	if _, err = c.ContainerStart(context.Background(), res.ID, client.ContainerStartOptions{}); err != nil {
		return err
	}

	// Wait for the command to finish
	wait := c.ContainerWait(context.Background(), res.ID, client.ContainerWaitOptions{
		Condition: container.WaitConditionNotRunning,
	})
	statusCh, errCh := wait.Result, wait.Error

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-statusCh:
		// Command finished successfully
	}

	return nil
}

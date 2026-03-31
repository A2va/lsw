package podman

import (
	"github.com/containers/podman/v6/libpod/define"
	"github.com/containers/podman/v6/pkg/bindings/containers"

	"github.com/A2va/lsw/pkg/config"
)

func New(name string) error {
	c, err := podmanClient()
	if err != nil {
		return err
	}

	spec, err := createSpec(config.Bottle{Name: name})
	if err != nil {
		return err
	}

	// We just need the container to start so Podman triggers the volume copy-up.
	spec.Command = []string{"true"}
	f := false
	spec.Terminal = &f
	spec.Stdin = &f

	res, err := containers.CreateWithSpec(c, &spec, &containers.CreateOptions{})
	if err != nil {
		return err
	}

	// Remove the container because all we wanted was to create a new volume
	// From there container creation are faster, so that we can mount new path easily be recreating the container (they are immutable)
	defer func() {
		containers.Remove(c, res.ID, new(containers.RemoveOptions).WithForce(true))
	}()

	err = containers.Start(c, res.ID, &containers.StartOptions{})
	if err != nil {
		return err
	}

	waitOpts := new(containers.WaitOptions).WithCondition([]define.ContainerStatus{
		define.ContainerStateExited,
		define.ContainerStateStopped,
	})

	_, err = containers.Wait(c, res.ID, waitOpts)
	if err != nil {
		return err
	}

	return nil
}

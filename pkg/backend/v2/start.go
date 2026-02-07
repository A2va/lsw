package v2

import (
	"fmt"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	"github.com/lxc/incus/shared/api"
)

func Start(bottle config.Bottle) error {
	log.Info("starting bottle (v2 backend)", "name", bottle.Name)
	c, err := incusClient()
	if err != nil {
		return fmt.Errorf("failed to connect to incus socket: %w", err)
	}

	op, err := c.UpdateInstanceState(bottle.Name, api.InstanceStatePut{Action: "start", Timeout: -1}, "")
	if err != nil {
		return fmt.Errorf("instance update failed: %w", err)
	}
	err = op.Wait()
	if err != nil {
		return fmt.Errorf("waiting operation failed: %w", err)
	}

	return nil
}

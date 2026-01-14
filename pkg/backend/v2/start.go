package v2

import (
	"fmt"

	"github.com/charmbracelet/log"
	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
)

func Start(name string) error {
	log.Debug("Start bottle on v2", "name", name)
	c, err := incus.ConnectIncusUnix("", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to incus socket: %w", err)
	}

	op, err := c.UpdateInstanceState(name, api.InstanceStatePut{Action: "start", Timeout: -1}, "")
	if err != nil {
		return fmt.Errorf("instance update failed: %w", err)
	}
	err = op.Wait()
	if err != nil {
		return fmt.Errorf("waiting operation failed: %w", err)
	}

	return nil
}

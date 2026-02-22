package v2

import (
	"fmt"
	"time"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
)

func waitForWindowsAgent(c incus.InstanceServer, vmName string) error {
	log.Debug("waiting for windows agent to be ready", "vm", vmName)

	timeout := time.After(1 * time.Minute)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	// Harmless command to verify the agent is accepting Exec requests
	cmd := []string{"cmd.exe", "/c", "echo", "ready"}

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for incus agent on %s", vmName)
		case <-ticker.C:
			_, err := runIncusCommand(c, vmName, cmd)
			if err == nil {
				log.Debug("windows agent is now ready")
				return nil
			}
			// You can ignore the error here, it usually means the agent is still booting.
			// log.Debug("agent not ready yet", "error", err)
		}
	}
}

func Start(bottle *config.Bottle) error {
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

	err = waitForWindowsAgent(c, bottle.Name)
	if err != nil {
		return fmt.Errorf("failed while waiting for windows to boot: %w", err)
	}

	for _, mount := range bottle.Mounts {
		mountFolder(c, bottle.Name, mount, "")
	}

	return nil
}

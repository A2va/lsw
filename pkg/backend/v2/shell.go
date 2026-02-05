package v2

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	incus "github.com/lxc/incus/client"
)

func addSharedDevice(bottle config.Bottle, cwd string, c incus.InstanceServer) string {
	inst, etag, err := c.GetInstance(bottle.Name)
	if err != nil {
		log.Fatal("get instance failed", "err", err)
	}

	d, exist := inst.Devices["shared"]
	if exist {
		return d["source"]
	}

	inst.Devices["shared"] = map[string]string{
		"type":   "disk",
		"source": cwd,
		"path":   "shared",
	}

	op, err := c.UpdateInstance(bottle.Name, inst.Writable(), etag)
	if err != nil {
		log.Fatal("update instance failed", "err", err)
	}
	err = op.Wait()
	if err != nil {
		log.Fatal("waiting operation failed", "err", err)
	}

	return ""
}

func Shell(bottle config.Bottle) error {
	// TODO Maybe start if stopped

	c, err := incusClient()
	if err != nil {
		return fmt.Errorf("failed to connect to incus socket: %w", err)
	}

	state, _, err := c.GetInstanceState(bottle.Name)
	if err != nil {
		return fmt.Errorf("failed to get instance state: %w", err)
	}

	if state.Status != "Running" {
		return fmt.Errorf("instance is %s, not Running", state.Status)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	source := addSharedDevice(bottle, cwd, c)

	var idAddr string
	for _, net := range state.Network {
		for _, addr := range net.Addresses {
			log.Debug("found adr", "family", addr.Family, "value", addr.Address, "scope", addr.Scope)
			// Family is "inet" for IPv4 or "inet6" for IPv6
			// Scope is "global" for external IPs
			if (addr.Family == "inet" || addr.Family == "inet6") && addr.Scope == "global" {
				idAddr = addr.Address
				break
			}
		}
	}

	username := os.Getenv("USER")

	cmd := exec.Command("ssh", username+"@"+idAddr,
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
	)
	cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	lsw, err := os.Executable()
	if err != nil {
		return err
	}
	log.Debug(lsw)

	// SSH cannot accept password from the cmd line, the only way is with a ask pass script.
	// Another solution would be to generate a SSH key and pack it with the unattended iso.
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SSH_ASKPASS=%s", lsw),
		"SSH_ASKPASS_REQUIRE=force",
		fmt.Sprintf("LSW_ASKPASS=%s", bottle.Password),
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to exex ssh: %w", err)
	}

	if source != "" && (source == cwd) {
		removeDevices(c, bottle.Name, []string{"shared"})
	}

	return nil
}

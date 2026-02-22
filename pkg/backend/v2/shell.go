package v2

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
)

func Shell(bottle *config.Bottle) error {
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

	mountPoint, err := mountFolder(c, bottle.Name, cwd, "cwd")
	if err != nil {
		return fmt.Errorf("failed to add shared device: %w", err)
	}

	if mountPoint.hostPath != "" && (mountPoint.hostPath == cwd) {
		defer func() {
			unmountFolder(c, bottle.Name, cwd, "cwd")
		}()
	}

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

	// ssh -t user@xxx.xxx.xxx.xxx "cd /d C:\directory_wanted & cmd /k"
	// ssh -t user@xxx.xxx.xxx.xxx "cd C:\directory_wanted ; powershell -NoExit"

	cmd := exec.Command("ssh", "-t", username+"@"+idAddr,
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("cd %s: ; powershell -NoExit", mountPoint.volumeLetter),
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
		return fmt.Errorf("failed to exec ssh: %w", err)
	}

	return nil
}

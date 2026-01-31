package v2

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/A2va/lsw/pkg/backend"
	"github.com/A2va/lsw/pkg/config"
	"github.com/charmbracelet/log"
	incus "github.com/lxc/incus/client"
)

func helperScript(bottle config.Bottle) (string, error) {
	cache, err := backend.GetCacheDir()
	if err != nil {
		return "", err
	}

	tmpDir := path.Join(cache, "tmp")
	log.Debug(tmpDir)

	wrapperScript := filepath.Join(tmpDir, "askpass_wrapper.sh")
	scriptContent := fmt.Sprintf("#!/bin/sh\necho -n %s\n", bottle.Password)
	err = os.WriteFile(wrapperScript, []byte(scriptContent), 0700)
	if err != nil {
		return wrapperScript, err
	}

	return wrapperScript, nil
}

func Shell(bottle config.Bottle) error {
	// TODO Maybe start if stopped

	log.Debug("test")
	c, err := incus.ConnectIncusUnix("", nil)
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

	var idAddr string

	for _, net := range state.Network {
		for _, addr := range net.Addresses {
			// Family is "inet" for IPv4 or "inet6" for IPv6
			// Scope is "global" for external IPs
			// FIXME add support for ipv6
			if addr.Family == "inet" && addr.Scope == "global" {
				idAddr = addr.Address
				break
			}
		}
	}

	username := os.Getenv("USER")

	// StrictHostKeyChecking is available from OpenSSH 7.6+, so might need to come back later to it and add
	// -o StrictHostKeyChecking=no to support older version
	cmd := exec.Command("ssh", username+"@"+idAddr, "-o StrictHostKeyChecking=accept-new")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	wrapperScript, err := helperScript(bottle)
	if err != nil {
		return err
	}
	defer os.Remove(wrapperScript)

	log.Debug(wrapperScript)
	// SSH cannot accept password from the cmd line, the only way is with a ask pass script.
	// Another solution would be to generate a SSH key and pack it with the unattended iso.
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SSH_ASKPASS=%s", wrapperScript),
		"SSH_ASKPASS_REQUIRE=force",
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create iso")
	}

	return nil
}

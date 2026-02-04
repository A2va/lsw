package v2

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/util"
)

func generateISO(sourceDir string, output string, label string) error {
	if util.PathExists(path.Join(sourceDir, output)) {
		return nil
	}

	args := []string{"-o", output, "-J", "-R", "-V", label, "-input-charset", "utf-8"}

	binaries := []string{"mkisofs", "genisoimage", "xorriso"}
	var selectedBin string

	// Find the first available binary in the system PATH
	for _, bin := range binaries {
		if _, err := exec.LookPath(bin); err == nil {
			selectedBin = bin
			break
		}
	}

	if selectedBin == "" {
		return fmt.Errorf("No ISO creation tool found (xorriso, mkisofs, or genisoimage).")
	}

	finalArgs := []string{}
	// xorriso requires special emulation flags to use mkisofs-style arguments
	if selectedBin == "xorriso" {
		finalArgs = append(finalArgs, "-as", "mkisofs")
	}

	// Append the shared flags and the final source directory
	finalArgs = append(finalArgs, args...)
	finalArgs = append(finalArgs, sourceDir)

	cmd := exec.Command(selectedBin, finalArgs...)
	cmd.Dir = sourceDir
	// TODO Log this
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	log.Debug("executing", "bin", selectedBin, "args", finalArgs)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create iso")
	}

	return nil
}

func splitStrict(content string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		if len(key) == 0 || key[0] == '#' {
			continue
		}
		// Overwrite previous key.
		value := parts[1]
		if len(value) > 2 && value[0] == '"' && value[len(value)-1] == '"' {
			// Not exactly 100% right but #closeenough. See for more details
			// https://www.freedesktop.org/software/systemd/man/os-release.html
			var err error
			value, err = strconv.Unquote(value)
			if err != nil {
				continue
			}
		}
		out[key] = value
	}
	return out
}

func readOSRelease() map[string]string {
	var osRelease map[string]string
	if bytes, err := os.ReadFile("/etc/os-release"); err == nil {
		osRelease = splitStrict(string(bytes))
	}
	return osRelease
}

func incusClient() (incus.InstanceServer, error) {
	_, exist := os.LookupEnv("INCUS_DIR")

	possibleSocketDir := []string{
		"/var/lib/incus",
		"/run/incus",
	}

	idx := 0

	// Check that the user did not set the variable so that we can set it ourselves
	if !exist {
		idLike := readOSRelease()["ID_LIKE"]
		if idLike == "fedora" {
			idx = 1
		} else {
			idx = 0
		}
		os.Setenv("INCUS_DIR", possibleSocketDir[idx])
	}

	c, err := incus.ConnectIncusUnix("", nil)
	if err != nil {
		// Not luck here the INCUS_DIR was set and it's failing
		if exist {
			return nil, err
		}
		// Try the other possible soket path before failing
		os.Setenv("INCUS_DIR", possibleSocketDir[1-idx])
		c, err := incus.ConnectIncusUnix("", nil)
		if err != nil {
			return nil, err
		}

		return c, nil
	}

	return c, nil

}

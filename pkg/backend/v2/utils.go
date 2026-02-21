package v2

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	incus "github.com/lxc/incus/client"
	"github.com/lxc/incus/shared/api"
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

// updateInstance applies changes to an Incus instance's configuration and devices.
// The modifyFn function receives the current instance object and should apply
// any desired changes to its Config and Devices fields.
func updateInstance(c incus.InstanceServer, vmName string, modifyFn func(*api.Instance) error) error {
	inst, etag, err := c.GetInstance(vmName)
	if err != nil {
		return fmt.Errorf("failed to fetch instance '%s': %w", vmName, err)
	}

	if err := modifyFn(inst); err != nil {
		return fmt.Errorf("failed to apply modifications to instance '%s': %w", vmName, err)
	}

	op, err := c.UpdateInstance(vmName, inst.Writable(), etag)
	if err != nil {
		return fmt.Errorf("failed to update instance '%s': %w", vmName, err)
	}
	if err := op.Wait(); err != nil {
		return fmt.Errorf("waiting for instance '%s' update operation failed: %w", vmName, err)
	}
	return nil
}

func addDevices(c incus.InstanceServer, vmName string, devicesToAdd map[string]map[string]string) error {
	return updateInstance(c, vmName, func(inst *api.Instance) error {
		maps.Copy(inst.Devices, devicesToAdd)
		return nil
	})
}

func removeDevices(c incus.InstanceServer, vmName string, devicesToRemove []string) error {
	return updateInstance(c, vmName, func(inst *api.Instance) error {
		for _, device := range devicesToRemove {
			_, exist := inst.Devices[device]
			if exist {
				delete(inst.Devices, device)
			}
		}
		return nil
	})
}

func runIncusCommand(c incus.InstanceServer, name string, cmd []string) error {
	log.Debug("runnning incus command", "cmd", cmd)

	req := api.InstanceExecPost{
		Command:     cmd,
		WaitForWS:   true,
		Interactive: false,
	}

	op, err := c.ExecInstance(name, req, nil)
	if err != nil {
		return err
	}
	return op.Wait()
}

// Based on https://github.com/virtio-win/kvm-guest-drivers-windows/wiki/Virtiofs:-Shared-file-system

// Name is optional
// Return the path of the mounted folder if exists
func mountFolder(c incus.InstanceServer, vmName string, path string, name string) (string, error) {
	if name == "" {
		h := sha256.New()
		h.Write([]byte(path))
		bs := h.Sum(nil)[:7]
		name = string(bs)
	}

	log.Debug("mount folder", "name", name, "path", path)

	inst, etag, err := c.GetInstance(vmName)
	if err != nil {
		return "", fmt.Errorf("failed to get instance for adding shared device: %w", err)
	}

	d, exist := inst.Devices[name]
	if exist {
		return d["source"], nil
	}

	// Add the device to the instance
	inst.Devices[name] = map[string]string{
		"type":   "disk",
		"source": path,
		"path":   name,
	}

	op, err := c.UpdateInstance(vmName, inst.Writable(), etag)
	if err != nil {
		return "", fmt.Errorf("failed to update instance to add shared device: %w", err)
	}
	if err := op.Wait(); err != nil {
		return "", fmt.Errorf("waiting for add shared device operation failed: %w", err)
	}

	time.Sleep(100 * time.Millisecond)

	// FIXME the volume letter needs to be changed
	cmd := []string{
		`C:\Program Files (x86)\WinFsp\bin\launchctl-x64.exe`,
		"start",
		"virtiofs",
		"viofsZ",
		fmt.Sprintf("incus_%s", name),
		"Z:",
	}

	err = runIncusCommand(c, vmName, cmd)
	if err != nil {
		return "", err
	}
	return path, nil
}

func unmountFolder(c incus.InstanceServer, vmName string, path string, name string) error {
	if name == "" {
		h := sha256.New()
		h.Write([]byte(path))
		bs := h.Sum(nil)[:7]
		name = string(bs)
	}

	log.Debug("unmount folder", "name", name, "path", path)
	cmd := []string{
		`C:\Program Files (x86)\WinFsp\bin\launchctl-x64.exe`,
		"stop",
		"virtiofs",
		"viofsZ",
	}

	err := runIncusCommand(c, vmName, cmd)
	if err != nil {
		return err
	}

	time.Sleep(200 * time.Millisecond)

	inst, _, err := c.GetInstance(vmName)
	if err != nil {
		return fmt.Errorf("failed to get instance for adding shared device: %w", err)
	}

	_, exist := inst.Devices[name]
	if exist {
		return removeDevices(c, vmName, []string{name})
	}

	return nil
}

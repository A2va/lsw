package v2

import (
	"fmt"
	"os/exec"
	"path"

	"github.com/charmbracelet/log"
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

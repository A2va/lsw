package backend

import (
	"github.com/A2va/lsw/pkg/config"
	"github.com/A2va/lsw/pkg/utils"
	"github.com/charmbracelet/log"
)

func GetBottle(name string) (*config.Bottle, bool) {
	if name == "" {
		return nil, false
	}

	cfg := config.Get()

	var bottleName string
	if len(name) >= 1 {
		bottleName = name
	} else {
		bottleName = cfg.DefaultBottle
	}

	log.Info("retrieving bottle", "name", bottleName)

	for i := range cfg.Bottles {
		if cfg.Bottles[i].Name == bottleName {
			log.Debug("bottle", "value", &cfg.Bottles[i], "found", true)
			return &cfg.Bottles[i], true
		}
	}

	log.Debug("bottle", "found", false)
	return nil, false
}

func GetShell(bottle config.Bottle) string {
	shell := bottle.Shell
	if shell == "" {
		shell = config.Get().DefaultShell
	}

	// Define mappings for version-specific overrides
	var overrides map[string]string
	switch bottle.Version {
	case "v1":
		overrides = map[string]string{"powershell": "pwsh", "pwsh": "pwsh", "cmd": "cmd"}
	case "v2":
		overrides = map[string]string{"powershell": "powershell", "pwsh": "powershell", "cmd": "cmd"}
	}

	// Apply override only if the shell exists in the map
	if mapped, ok := overrides[shell]; ok {
		return mapped
	}

	utils.Panic("shell is not support")
	return shell
}

package config

import (
	"charm.land/log/v2"
)

// A bottle is a single unit for representing a single instance of windows with it's specific set of congiuration and software
// Can be tied to a v1 or v2 backend.
type Bottle struct {
	Name string `toml:"name"`
	// v1 or v2
	Version    string `toml:"version"`
	Shell      string `toml:"shell"`
	V1Provider string `toml:"v1_provider"`
	// Permanently mounted folder (absolute path)
	Mounts []string
	// the plain text password for a v2 vm
	Password string
}

type BottleStatus struct {
	Name    string
	Running bool
	// Current working directory when lsw shell was executed
	EnteredFrom string
}

func GetBottle(name string) (*Bottle, bool) {
	cfg := Get()

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

func (b *Bottle) GetShell() string {
	shell := b.Shell
	if shell == "" {
		shell = Get().DefaultShell
	}

	// Define mappings for version-specific overrides
	var overrides map[string]string
	switch b.Version {
	case "v1":
		overrides = map[string]string{"powershell": "pwsh", "pwsh": "pwsh", "cmd": "cmd"}
	case "v2":
		overrides = map[string]string{"powershell": "powershell", "pwsh": "powershell", "cmd": "cmd"}
	}

	// Apply override only if the shell exists in the map
	if mapped, ok := overrides[shell]; ok {
		return mapped
	}

	log.Warn("falling back to cmd prompt")
	return "cmd"
}

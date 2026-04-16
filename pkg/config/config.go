package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	"github.com/BurntSushi/toml"
)

var cfg Config
var v Version

type Version struct {
	Version     string
	Commit      string
	ShortCommit string
	Date        string
	DebugFlag   bool
}

type Config struct {
	Bottles        []Bottle `toml:"bottles"`
	DefaultBackend string   `toml:"default_backend"`
	DefaultBottle  string   `toml:"default_bottle"`
	DefaultShell   string   `toml:"default_shell"`
	// Store the first found provider in case a user install another later
	DefaultV1Provider string `toml:"default_v1_provider"`
}

func getConfigPath() (string, error) {
	c, exist := os.LookupEnv("XDG_CONFIG_HOME")

	if exist {
		return filepath.Join(c, "lsw", "config.toml"), nil
	}

	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return "", homeErr
	}

	return filepath.Join(home, ".config", "lsw", "config.toml"), nil
}

// Load the config and fill default values if non existent
func CheckAndLoad() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	confDir := filepath.Dir(configPath)

	if err := os.MkdirAll(confDir, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("Error creating config directory [%v]", err)
	}

	log.Info("config directory is", "dir", confDir)
	f, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE, 0664)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	defer f.Close()

	cfg.Bottles = []Bottle{}
	cfg.DefaultBackend = "v2"
	cfg.DefaultV1Provider = ""
	cfg.DefaultShell = "powershell"

	if _, err := toml.NewDecoder(f).Decode(&cfg); err != nil {
		return err
	}

	if cfg.Bottles == nil {
		cfg.Bottles = []Bottle{}
	}
	return nil
}

func Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(configPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0664)
	if err != nil {
		return err
	}

	defer f.Close()

	encoder := toml.NewEncoder(f)
	err = encoder.Encode(cfg)
	if err != nil {
		return err
	}

	return nil
}

func Get() *Config {
	return &cfg
}

func (c *Config) RemoveBottle(name string) {
	for i, b := range c.Bottles {
		if b.Name == name {
			c.Bottles = append(c.Bottles[:i], c.Bottles[i+1:]...)
			break
		}
	}

	if c.DefaultBottle == name {
		c.DefaultBottle = ""
	}
}

func (c *Config) AddBottle(bottle Bottle) {
	c.Bottles = append(c.Bottles, bottle)
	// Set the default bottle if not already set
	if cfg.DefaultBottle == "" {
		cfg.DefaultBottle = bottle.Name
	}
}

func SetVersion(versionCmd string, debugFlag bool) {
	fullVersion := versionCmd
	parts := strings.Split(fullVersion, "\n")

	// If the executable is not built by goreleaser versionCmd will contain just dev
	// so parts will have a size of 1
	if len(parts) == 1 || versionCmd == "" {
		v = Version{Version: "dev", Commit: "dev", ShortCommit: "dev", DebugFlag: debugFlag}
		return
	}

	version := parts[0]
	if version == "" {
		version = "dev"
	}

	var commit string
	var shortCommit string
	if len(parts) > 1 && strings.HasPrefix(parts[1], "commit: ") {
		commit = strings.TrimPrefix(parts[1], "commit: ")
		shortCommit = commit[:7]
	}

	var date string
	if len(parts) > 1 && strings.HasPrefix(parts[2], "date: ") {
		date = strings.TrimPrefix(parts[2], "date: ")
	}

	v = Version{Version: version, Commit: commit, ShortCommit: shortCommit, Date: date, DebugFlag: debugFlag}
}

func GetVersion() Version {
	return v
}

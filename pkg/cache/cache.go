package cache

import (
	"os"
	"path/filepath"
)

func GetCacheDir() (string, error) {
	c, exist := os.LookupEnv("XDG_CACHE_HOME")

	if exist {
		return filepath.Join(c, "lsw"), nil
	}

	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return "", homeErr
	}

	return filepath.Join(home, ".cache", "lsw"), nil
}

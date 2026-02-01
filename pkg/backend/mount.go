package backend

import (
	"path/filepath"

	"github.com/A2va/lsw/pkg/config"
)

func Mount(bottle config.Bottle, folder string) error {
	// TODO Mabye resolve symlink

	absFolder, err := filepath.Abs(folder)
	if err != nil {
		return err
	}
	bottle.Mounts = append(bottle.Mounts, absFolder)
	return nil
}

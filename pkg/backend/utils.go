package backend

import (
	"github.com/A2va/lsw/pkg/config"
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

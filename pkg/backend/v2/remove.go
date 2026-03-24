package v2

import (
	"github.com/A2va/lsw/pkg/config"
	"charm.land/log/v2"
)

func Remove(bottle *config.Bottle) error {
	c, err := incusClient()
	if err != nil {
		return err
	}

	op, err := c.DeleteInstance(bottle.Name)
	if err != nil {
		return err
	}

	err = op.Wait()
	if err != nil {
		return err
	}
	log.Debug("remove bottle", "bottle", bottle.Name)

	config.Get().RemoveBottle(bottle.Name)
	return nil
}

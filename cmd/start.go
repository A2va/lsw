package cmd

import (
	v2 "github.com/A2va/lsw/pkg/backend/v2"
	"github.com/A2va/lsw/pkg/config"
	"github.com/spf13/cobra"
)

func startCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "start",
		Short:         "Start a bottle",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Get()

			var bottleName string
			if len(args) >= 1 {
				bottleName = args[0]
			} else {
				bottleName = cfg.DefaultBottle
			}

			var bottle config.Bottle
			for _, b := range cfg.Bottles {
				if b.Name == bottleName {
					bottle = b
					break
				}
			}

			if bottle.Version == "v2" {
				return v2.Start(bottleName)
			}
			return nil
		},
	}
	return cmd
}

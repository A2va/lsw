package cmd

import (
	"fmt"

	v1 "github.com/A2va/lsw/pkg/backend/v1"
	v2 "github.com/A2va/lsw/pkg/backend/v2"
	"github.com/A2va/lsw/pkg/config"
	"github.com/spf13/cobra"
)

func removeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "remove [bottle-name]",
		Aliases:       []string{"rm"},
		Short:         "Remove a Windows bottle",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			bottleName := ""
			if len(args) >= 1 {
				bottleName = args[0]
			}
			bottle, found := config.GetBottle(bottleName)

			if !found {
				return fmt.Errorf("bottle '%s' not found", bottleName)
			}

			if bottle.Version == "v1" {
				return v1.Remove(bottle)
			} else if bottle.Version == "v2" {
				return v2.Remove(bottle)
			}
			return fmt.Errorf("invalid backend version: %s", bottle.Version)
		},
	}
	return cmd
}

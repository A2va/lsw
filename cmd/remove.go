package cmd

import (
	"fmt"

	"github.com/A2va/lsw/pkg/backend"
	v1 "github.com/A2va/lsw/pkg/backend/v1"
	v2 "github.com/A2va/lsw/pkg/backend/v2"
	"github.com/spf13/cobra"
)

func removeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Short:   "Remove a bottle",
		Long: `Can specify the bottle name, or LSW will use the default configured bottle.

Example:
  lsw remove my-windows-bottle
  lsw remove # Removes the default configured bottle`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			bottleName := ""
			if len(args) >= 1 {
				bottleName = args[0]
			}
			bottle, found := backend.GetBottle(bottleName)

			if !found {
				return fmt.Errorf("not found the bottle")
			}

			if bottle.Version == "v1" {
				return v1.Remove(bottle)
			} else if bottle.Version == "v2" {
				return v2.Remove(bottle)
			}
			return fmt.Errorf("not a valid backend")
		},
	}
	return cmd
}

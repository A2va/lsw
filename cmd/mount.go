package cmd

import (
	"fmt"

	"github.com/A2va/lsw/pkg/backend"
	"github.com/spf13/cobra"
)

func mountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "mount",
		Short:         "Add a mount directory to a bottle",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) <= 1 {
				return fmt.Errorf("missing bottle or directory arg")
			}

			bottleName := ""
			if len(args) >= 1 {
				bottleName = args[0]
			}
			bottle, found := backend.GetBottle(bottleName)

			if !found {
				return fmt.Errorf("not found the bottle")
			}

			folder := args[1]

			return backend.Mount(bottle, folder)
		},
	}
	return cmd
}

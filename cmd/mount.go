package cmd

import (
	"fmt"

	"github.com/A2va/lsw/pkg/backend"
	"github.com/spf13/cobra"
)

func mountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mount [bottle-name] <directory-path>",
		Short: "Mount a host directory into a Windows bottle",
		Long: `Mount a directory from your Linux host into a specified Windows bottle.

You can specify the bottle name, or LSW will use the default configured bottle.

Notes:
  - For v2 bottles (Incus VM), automatic mounting usually occurs from the current working directory during 'lsw shell'.
  - For v1 bottles, restart the current shell session for mounts to take effect.

Example:
  lsw mount my-windows-bottle ~/my-project
  lsw mount ~/my-project # Mounts into the default configured bottle`,
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

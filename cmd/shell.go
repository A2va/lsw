package cmd

import (
	"fmt"

	"github.com/A2va/lsw/pkg/backend"
	v1 "github.com/A2va/lsw/pkg/backend/v1"
	v2 "github.com/A2va/lsw/pkg/backend/v2"
	"github.com/spf13/cobra"
)

func runShellCommand(cmd *cobra.Command, args []string) error {
	bottleName := ""
	if len(args) >= 1 {
		bottleName = args[0]
	}
	bottle, found := backend.GetBottle(bottleName)

	if !found {
		return fmt.Errorf("not found the bottle")
	}

	if bottle.Version == "v1" {
		return v1.Shell(bottle)
	} else if bottle.Version == "v2" {
		return v2.Shell(bottle)
	}
	return fmt.Errorf("not a valid backend")
}

func shellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "shell [bottle-name]",
		Aliases: []string{"s"},
		Short:   "Enter an interactive shell into a Windows bottle",
		Long: `Can specify the bottle name, or LSW will use the default configured bottle.

Example:
  lsw shell my-windows-bottle
  lsw shell # Uses the default configured bottle`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runShellCommand,
	}
	return cmd
}

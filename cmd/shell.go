package cmd

import (
	"fmt"

	v1 "github.com/A2va/lsw/pkg/backend/v1"
	v2 "github.com/A2va/lsw/pkg/backend/v2"
	"github.com/A2va/lsw/pkg/config"
	"github.com/spf13/cobra"
)

func runShellCommand(cmd *cobra.Command, args []string) error {
	bottleName := ""
	if len(args) >= 1 {
		bottleName = args[0]
	}
	bottle, found := config.GetBottle(bottleName)

	if !found {
		return fmt.Errorf("bottle '%s' not found", bottleName)
	}

	cmdFlag, _ := cmd.Flags().GetString("cmd")

	if bottle.Version == "v1" {
		return v1.Shell(bottle, cmdFlag)
	} else if bottle.Version == "v2" {
		return v2.Shell(bottle, cmdFlag)
	}
	return fmt.Errorf("not a valid backend")
}

func shellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "shell [bottle-name]",
		Aliases:       []string{"s"},
		Short:         "Enter an interactive shell in a Windows bottle",
		Long:          `Specify a bottle name, or use the default bottle if configured.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runShellCommand,
	}

	cmd.Flags().String("cmd", "", "Execute the given cmd in Command Prompt (no interaction)")
	return cmd
}

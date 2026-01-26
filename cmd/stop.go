package cmd

import (
	"fmt"

	"github.com/A2va/lsw/pkg/backend"
	v1 "github.com/A2va/lsw/pkg/backend/v1"
	v2 "github.com/A2va/lsw/pkg/backend/v2"
	"github.com/spf13/cobra"
)

func stopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "stop",
		Short:         "Stop a running bottle",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			bottle, found := backend.GetBottle(args[0])

			if !found {
				return fmt.Errorf("not found the bottle")
			}

			if bottle.Version == "v2" {
				return v2.Stop(bottle)
			} else if bottle.Version == "v1" {
				return v1.Stop(bottle)
			}
			return nil
		},
	}
	return cmd
}

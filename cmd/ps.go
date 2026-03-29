package cmd

import (
	"github.com/A2va/lsw/pkg/backend"
	"github.com/spf13/cobra"
)

func psCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ps",
		Short:         "List Bottles",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			backend.Ps()
			return nil
		},
	}
	return cmd
}

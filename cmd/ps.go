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
			noHeaders, _ := cmd.Flags().GetBool("noheading")
			all, _ := cmd.Flags().GetBool("all")
			backend.Ps(noHeaders, all)
			return nil
		},
	}

	cmd.PersistentFlags().Bool("noheading", false, "Do not print headers")
	cmd.PersistentFlags().BoolP("all", "a", false, "Show all the bottles, default is only running bottles")
	return cmd
}

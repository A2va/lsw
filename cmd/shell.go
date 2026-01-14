package cmd

import (
	"fmt"

	v2 "github.com/A2va/lsw/pkg/backend/v2"
	"github.com/A2va/lsw/pkg/config"
	log "github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

func shellCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "shell",
		Aliases:       []string{"s"},
		Short:         "Enter a windows shell",
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
			found := false

			for _, b := range cfg.Bottles {
				if b.Name == bottleName {
					bottle = b
					found = true
					break
				}
			}

			log.Debug("found", "found", found)
			if !found {
				return fmt.Errorf("not found the bottle")
			}

			askpass, _ := cmd.Flags().GetBool("askpass")
			if askpass {
				fmt.Print(bottle.Password)
				return nil
			}

			if bottle.Version == "v2" {
				return v2.Shell(bottle)
			}
			return nil
		},
	}
	cmd.PersistentFlags().Bool("askpass", false, "Used for an SSH connection")

	return cmd
}

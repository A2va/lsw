package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	v1 "github.com/A2va/lsw/pkg/backend/v1"
	v2 "github.com/A2va/lsw/pkg/backend/v2"
	"github.com/A2va/lsw/pkg/config"
	log "github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// Give a new name to a bottle
func newName(names []string) string {
	if len(names) == 0 {
		return "win"
	}

	extractNum := func(name string) int {
		s := strings.Split(name, "-")

		if len(s) >= 2 {
			n, err := strconv.ParseInt(s[1], 10, 0)
			if err != nil {
				log.Fatal(err)
			}
			return int(n)
		} else {
			return 0
		}
	}
	num := 0

	for idx, name := range names {
		n := extractNum(name)
		if n != idx {
			num = idx
			break
		}
	}

	if num == 0 {
		return fmt.Sprintf("win-%d", len(names))
	} else {
		return fmt.Sprintf("win-%d", num)
	}
}

func autoName(cfg *config.Config) string {
	names := make([]string, 0, len(cfg.Bottles))
	for _, b := range cfg.Bottles {
		if strings.HasPrefix(b.Name, "win") {
			names = append(names, b.Name)
		}
	}

	sort.Slice(names, func(i, j int) bool {
		if names[i] < names[j] {
			return true
		}
		return false
	})
	return newName(names)
}

func newV1Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "v1",
		Short: "Create a bottle using v1 backend",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Get()

			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				name = autoName(cfg)
			}
			log.Debug("auto naming", "name", name)

			init, _ := cmd.Flags().GetBool("init")
			if init {
				v1.Init()
				return nil
			}

			return v1.New(name)
		},
	}

	// v1-specific flags (if any)

	return cmd
}

func newV2Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "v2",
		Short: "Create a bottle using v2 backend",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Get()

			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				name = autoName(cfg)
			}

			init, _ := cmd.Flags().GetBool("init")
			if init {
				v2.Init()
				return nil
			}

			log.Debug("auto naming", "name", name)

			ram, _ := cmd.Flags().GetString("ram")
			disk, _ := cmd.Flags().GetString("disk")
			cpus, _ := cmd.Flags().GetString("cpus")
			password, _ := cmd.Flags().GetString("password")
			user, _ := cmd.Flags().GetString("user")

			return v2.New("amd64", v2.NewArgument{
				Name:     name,
				Ram:      ram,
				Disk:     disk,
				Cpus:     cpus,
				Password: password,
				Username: user,
			})
		},
	}

	// v2-only flags
	cmd.Flags().String("ram", "6GiB", "Define the RAM in GiB for the VM, only for v2 backend.")
	cmd.Flags().String("disk", "25GiB", "Define the disk space in GiB for the VM, only for v2 backend.")
	cmd.Flags().String("cpus", "4", "Set the number of cpu cores for the VM, only for v2 backend.")

	cmd.Flags().String("password", "123456", "User password")
	cmd.Flags().String("user", os.Getenv("USER"), "Username")
	return cmd
}

func newCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "new",
		Aliases:       []string{"n"},
		Short:         "Create a new bottle. If you want to create a bottle without requiring an internet connection, first pass the init flag.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().Bool("init", false, "Download any necessary files then exit without creating a bottle")
	cmd.PersistentFlags().StringP("name", "n", "", "Define a name for the bottle, if not provided an automatic name will be given.")

	cmd.AddCommand(newV2Cmd())
	cmd.AddCommand(newV1Cmd())

	return cmd
}

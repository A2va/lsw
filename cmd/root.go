package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/A2va/lsw/pkg/backend"
	"github.com/A2va/lsw/pkg/config"
	log "github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"gopkg.in/natefinch/lumberjack.v2"
)

func initLog(debug bool) {
	logdir, err := backend.GetCacheDir()
	if err != nil {
		log.Fatalf("cannot get cache directory: %v", err)
	}

	logPath := filepath.Join(logdir, "logs", "lsw.log")

	fileLogger := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    2, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	}

	if debug {
		log.SetLevel(log.DebugLevel)
		log.SetOutput(io.MultiWriter(os.Stderr, fileLogger))
	} else {
		log.SetLevel(log.DebugLevel)
		log.SetOutput(fileLogger)
	}

	log.SetReportTimestamp(true)
	log.SetReportCaller(true)

	log.SetTimeFormat(time.DateTime)
}

func Execute(version string, exit func(int), args []string) {
	newRootCmd(version, exit).Execute(args)
}

func (cmd *rootCmd) Execute(args []string) {
	cmd.cmd.SetArgs(args)

	if err := cmd.cmd.Execute(); err != nil {
		code := 1
		msg := "command failed"
		if eerr, ok := err.(*exitError); ok {
			code = eerr.code
			if eerr.details != "" {
				msg = eerr.details
			}
		}
		log.Error(msg, "err", err)
		fmt.Printf("%s, err: %s\n", msg, err)
		cmd.exit(code)
	}
}

type rootCmd struct {
	cmd   *cobra.Command
	debug bool
	exit  func(int)
}

func newRootCmd(version string, exit func(int)) *rootCmd {
	root := &rootCmd{
		exit: exit,
	}
	cmd := &cobra.Command{
		Use:   "lsw",
		Short: "Manage isolated Windows environments (bottles) from Linux",
		Long: `LSW (Linux Subsystem for Windows) manages isolated Windows environments ("bottles") from Linux.

Features:
  - Supports v1 (Wine-based containers) and v2 (Incus VMs).
  - Create, access (shell), start, and stop Windows environments.
  - Facilitates cross-platform development and testing.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			_, err := backend.CreateAllCacheDirectories()
			if err != nil {
				log.Fatal("error creating cache directories", "err", err)
			}
			initLog(root.debug)

			// check and load config after handlers are configured
			err = config.CheckAndLoad()
			if err != nil {
				log.Fatal("error loading config file", "err", err)
			}

			config.SetVersion(cmd.Version, root.debug)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			err := config.Save()
			if err != nil {
				log.Error("error loading config file", "err", err)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runShellCommand(cmd, args)
			}
			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&root.debug, "debug", false, "Enable debug mode.")

	cmd.AddCommand(newCmd())
	cmd.AddCommand(shellCmd())
	cmd.AddCommand(startCmd())
	cmd.AddCommand(stopCmd())
	cmd.AddCommand(mountCmd())

	root.cmd = cmd
	return root
}

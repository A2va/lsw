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
		log.Fatalf("Cannot get cache directory: %v", err)
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
		log.SetOutput(fileLogger)
	}

	log.SetTimeFormat(time.Kitchen)
	log.SetReportTimestamp(true)
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
		Use:           "lsw",
		Short:         "WSL like",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			_, err := backend.CreateAllCacheDirectories()
			if err != nil {
				log.Fatal("Error creating cache directories", "err", err)
			}
			initLog(root.debug)

			// check and load config after handlers are configured
			err = config.CheckAndLoad()
			if err != nil {
				log.Fatal("Error loading config file", "err", err)
			}

			config.SetVersion(cmd.Version)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			err := config.Save()
			if err != nil {
				log.Error("Error loading config file", "err", err)
			}
		},
	}

	cmd.PersistentFlags().BoolVar(&root.debug, "debug", false, "Enable debug mode")

	cmd.AddCommand(newCmd())
	cmd.AddCommand(shellCmd())
	root.cmd = cmd
	return root
}

package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	log "charm.land/log/v2"
	"github.com/A2va/lsw/pkg/cache"
	"github.com/A2va/lsw/pkg/config"
	"github.com/A2va/lsw/pkg/utils"
	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
	"gopkg.in/natefinch/lumberjack.v2"
)

var ansiRegex = regexp.MustCompile("[\u001b\u009b][\\[()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]")

type splitWriter struct {
	tty  *os.File
	file io.Writer
}

// Fd exposes the terminal's file descriptor. This allows charmbracelet/log
// to correctly detect terminal support and enable colors natively.
func (w *splitWriter) Fd() uintptr {
	return w.tty.Fd()
}

func (w *splitWriter) Write(p []byte) (n int, err error) {
	n, err = w.tty.Write(p)
	if err != nil {
		return n, err
	}

	// Strip the ANSI color codes
	cleanBytes := ansiRegex.ReplaceAll(p, []byte(""))

	// Write the cleaned bytes to the log file
	_, err = w.file.Write(cleanBytes)
	if err != nil {
		return n, err
	}

	return len(p), nil
}

func initLog(debug bool) {
	logdir, err := cache.GetCacheDir()
	if err != nil {
		utils.Panic("cannot get cache directory", err)
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
		log.SetOutput(&splitWriter{
			tty:  os.Stderr,
			file: fileLogger,
		})
	} else {
		log.SetLevel(log.InfoLevel)
		log.SetOutput(fileLogger)
		log.SetColorProfile(colorprofile.NoTTY)
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
			err := cache.Init()
			if err != nil {
				utils.Panic("error creating cache directories", err)
			}
			initLog(root.debug)

			// check and load config after handlers are configured
			err = config.CheckAndLoad()
			if err != nil {
				utils.Panic("error loading config file", err)
			}

			config.SetVersion(cmd.Version, root.debug)
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			err := config.Save()
			if err != nil {
				log.Error("error loading config file", "err", err)
			}

			err = cache.Prune(1, 20)
			if err != nil {
				log.Warnf("error to prune the cache: %w", err)
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
	cmd.AddCommand(removeCmd())
	cmd.AddCommand(mountCmd())
	cmd.AddCommand(psCmd())

	root.cmd = cmd
	return root
}

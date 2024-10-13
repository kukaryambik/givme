package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/listpaths"
	"github.com/kukaryambik/givme/pkg/logging"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	appName = "givme"
)

type CommandOptions struct {
	DotenvFile       string
	Exclusions       []string
	Image            string
	RegistryUsername string
	RegistryPassword string
	RegistryMirror   string
	Retry            int
	RootFS           string
	TarFile          string
	Workdir          string
	Eval             bool
}

var (
	logLevel     string
	logFormat    string
	logTimestamp bool

	opts CommandOptions
)

func init() {
	viper.SetEnvPrefix(appName) // Environment variables prefixed with GIVME_
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // Automatically bind environment variables

	addFlags()

	// Bind flags to environment variables
	viper.BindPFlags(RootCmd.PersistentFlags())

	// Add subcommands for snapshot, restore, and cleanup.
	RootCmd.AddCommand(
		cleanupCmd,
		exportCmd,
		getenvCmd,
		loadCmd,
		restoreCmd,
		saveCmd,
		snapshotCmd,
	)
}

var RootCmd = &cobra.Command{
	Use: appName,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set variables from flags or environment
		opts.Workdir = viper.GetString("workdir")
		opts.RootFS = viper.GetString("rootfs")
		opts.Exclusions = viper.GetStringSlice("exclude")
		opts.RegistryMirror = viper.GetString("registry-mirror")
		opts.RegistryUsername = viper.GetString("registry-username")
		opts.RegistryPassword = viper.GetString("registry-password")
		opts.Retry = viper.GetInt("retry")
		logLevel = viper.GetString("verbosity")

		// Set up logging
		if err := logging.Configure(logLevel, logFormat, logTimestamp, opts.Eval); err != nil {
			return err
		}

		// Check if rootfs and workdir are the same
		if opts.RootFS == opts.Workdir {
			return fmt.Errorf("rootfs and workdir cannot be the same")
		}

		// Ensure the work directory exists.
		if err := os.MkdirAll(opts.Workdir, 0755); err != nil {
			logrus.Fatalf("Error creating work directory: %v", err)
		}

		// Build exclusions
		if excl, err := listpaths.Excl(
			opts.RootFS, append(opts.Exclusions, opts.Workdir, "!"+opts.RootFS),
		); err != nil {
			return err
		} else {
			opts.Exclusions = excl
		}
		return nil
	},
}

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Create a snapshot archive",
	PreRun: func(cmd *cobra.Command, args []string) {
		if opts.TarFile == "" {
			opts.TarFile = filepath.Join(opts.Workdir, defaultSnapshotName+".tar")
		}
		if opts.DotenvFile == "" {
			opts.DotenvFile = filepath.Join(opts.Workdir, defaultSnapshotName+".env")
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return snapshot(&opts)
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore from a snapshot archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := restore(&opts)
		if err != nil && opts.Eval {
			fmt.Print("false")
		}
		return err
	},
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up directories",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cleanup(&opts)
	},
}

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save image to tar archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		_, err := save(&opts)
		return err
	},
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export container image tar and config",
	Args:  cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		return export(&opts)
	},
}

var getenvCmd = &cobra.Command{
	Use:   "getenv",
	Short: "Get container image environment variables",
	Args:  cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		err := getenv(&opts)
		if err != nil && opts.Eval {
			fmt.Print("false")
		}
		return err
	},
}

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Load container image tar and apply it to the system",
	Args:  cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		err := load(&opts)
		if err != nil && opts.Eval {
			fmt.Print("false")
		}
		return err
	},
}

func addFlags() {
	execdir := util.GetExecDir()

	RootCmd.PersistentFlags().StringVarP(
		&opts.RootFS, "rootfs", "R", "/", "RootFS directory; or use GIVME_ROOTFS")
	RootCmd.MarkPersistentFlagDirname("rootfs")

	RootCmd.PersistentFlags().StringVarP(
		&opts.Workdir, "workdir", "W", execdir, "Working directory; or use GIVME_WORKDIR")
	RootCmd.MarkPersistentFlagDirname("workdir")

	RootCmd.PersistentFlags().StringSliceVarP(
		&opts.Exclusions, "exclude", "X", nil, "Excluded directories; or use GIVME_EXCLUDE")

	RootCmd.PersistentFlags().BoolVarP(
		&opts.Eval, "eval", "e", false, "Output might be evaluated")

	RootCmd.PersistentFlags().StringVarP(
		&opts.TarFile, "tar-file", "f", "", "Path to the tar file")
	RootCmd.MarkPersistentFlagFilename("tar-file", ".tar")
	restoreCmd.MarkPersistentFlagRequired("tar-file")

	RootCmd.PersistentFlags().StringVarP(
		&opts.DotenvFile, "dotenv-file", "d", "", "Path to the .env file")
	RootCmd.MarkPersistentFlagFilename("dotenv-file", ".env")

	RootCmd.PersistentFlags().IntVar(
		&opts.Retry, "retry", 0, "Retry attempts of saving the image; or use GIVME_RETRY")

	RootCmd.PersistentFlags().StringVar(
		&opts.RegistryMirror, "registry-mirror", "",
		"Registry mirror; or use GIVME_REGISTRY_MIRROR",
	)

	RootCmd.PersistentFlags().StringVar(
		&opts.RegistryUsername, "registry-username", "",
		"Username for registry authentication; or use GIVME_REGISTRY_USERNAME",
	)

	RootCmd.PersistentFlags().StringVar(
		&opts.RegistryPassword, "registry-password", "",
		"Password for registry authentication; or use GIVME_REGISTRY_PASSWORD",
	)

	// Logging flags
	RootCmd.PersistentFlags().StringVarP(
		&logLevel, "verbosity", "v", logging.DefaultLevel,
		"Log level (trace, debug, info, warn, error, fatal, panic)",
	)
	RootCmd.PersistentFlags().StringVar(
		&logFormat, "log-format", logging.FormatColor,
		"Log format (text, color, json)",
	)
	RootCmd.PersistentFlags().BoolVar(
		&logTimestamp, "log-timestamp", logging.DefaultLogTimestamp,
		"Timestamp in log output",
	)
}

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
	// Command options
	Cmd              []string
	DotenvFile       string   `mapstructure:"dotenv-file"`
	Entrypoint       string   `mapstructure:"entrypoint"`
	Eval             bool     `mapstructure:"eval"`
	Exclusions       []string `mapstructure:"exclude"`
	Image            string
	RegistryMirror   string `mapstructure:"registry-mirror"`
	RegistryPassword string `mapstructure:"registry-password"`
	RegistryUsername string `mapstructure:"registry-username"`
	Retry            int    `mapstructure:"retry"`
	RootFS           string `mapstructure:"rootfs"`
	TarFile          string `mapstructure:"tar-file"`
	Workdir          string `mapstructure:"workdir"`

	// Logging
	LogLevel     string `mapstructure:"log-level"`
	LogFormat    string `mapstructure:"log-format"`
	LogTimestamp bool   `mapstructure:"log-timestamp"`
}

var (
	opts    CommandOptions
	execdir = util.GetExecDir()
)

func mkFlags(c func(*cobra.Command), l ...*cobra.Command) {
	for _, cmd := range l {
		c(cmd)
	}
}

func init() {
	addFlags()

	viper.SetEnvPrefix(appName) // Environment variables prefixed with GIVME_
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // Automatically bind environment variables

	// Bind flags to environment variables
	viper.BindPFlags(RootCmd.PersistentFlags())

	// Add subcommands for snapshot, restore, and cleanup.
	RootCmd.AddCommand(
		cleanupCmd,
		exportCmd,
		getenvCmd,
		loadCmd,
		prootCmd,
		restoreCmd,
		saveCmd,
		snapshotCmd,
	)
}

var RootCmd = &cobra.Command{
	Use: appName,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set variables from flags or environment
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %v", err)
		}
		viper.Unmarshal(&opts)

		// Set up logging
		if err := logging.Configure(opts.LogLevel, opts.LogFormat, opts.LogTimestamp, opts.Eval); err != nil {
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
			opts.RootFS, append(opts.Exclusions, opts.Workdir, execdir, "!"+opts.RootFS),
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
		_, err := load(&opts)
		if err != nil && opts.Eval {
			fmt.Print("false")
		}
		return err
	},
}

var prootCmd = &cobra.Command{
	Use:  "proot",
	Args: cobra.MinimumNArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		opts.Cmd = args[1:]
		return proot(&opts)
	},
}

func addFlags() {

	// Global flags
	RootCmd.PersistentFlags().StringVarP(
		&opts.RootFS, "rootfs", "R", "/", "RootFS directory; or use GIVME_ROOTFS")
	RootCmd.MarkPersistentFlagDirname("rootfs")

	RootCmd.PersistentFlags().StringVarP(
		&opts.Workdir, "workdir", "W", filepath.Join(execdir, "tmp"), "Working directory; or use GIVME_WORKDIR")
	RootCmd.MarkPersistentFlagDirname("workdir")

	RootCmd.PersistentFlags().StringSliceVarP(
		&opts.Exclusions, "exclude", "X", nil, "Excluded directories; or use GIVME_EXCLUDE")

	// Logging flags
	RootCmd.PersistentFlags().StringVarP(
		&opts.LogLevel, "verbosity", "v", logging.DefaultLevel,
		"Log level (trace, debug, info, warn, error, fatal, panic)",
	)
	RootCmd.PersistentFlags().StringVar(
		&opts.LogFormat, "log-format", logging.FormatColor,
		"Log format (text, color, json)",
	)
	RootCmd.PersistentFlags().BoolVar(
		&opts.LogTimestamp, "log-timestamp", logging.DefaultLogTimestamp,
		"Timestamp in log output",
	)

	// Subcommand flags

	// --eval
	mkFlags(
		func(cmd *cobra.Command) {
			cmd.Flags().BoolVarP(&opts.Eval, "eval", "e", false, "Output might be evaluated")
		},
		restoreCmd, loadCmd, getenvCmd,
	)

	// --tar-file
	mkFlags(
		func(cmd *cobra.Command) {
			cmd.Flags().StringVarP(
				&opts.TarFile, "tar-file", "f", "", "Path to the tar file")
			cmd.MarkFlagFilename("tar-file", ".tar")
		},
		snapshotCmd, saveCmd, prootCmd, exportCmd, restoreCmd,
	)
	restoreCmd.MarkFlagRequired("tar-file")

	// --dotenv-file
	mkFlags(
		func(cmd *cobra.Command) {
			cmd.Flags().StringVarP(
				&opts.DotenvFile, "dotenv-file", "d", "", "Path to the .env file")
			cmd.MarkFlagFilename("dotenv-file", ".env")
		},
		snapshotCmd, getenvCmd, restoreCmd,
	)

	// --retry and --registry-[mirror|username|password]
	mkFlags(
		func(cmd *cobra.Command) {
			cmd.Flags().IntVar(
				&opts.Retry, "retry", 0, "Retry attempts of saving the image; or use GIVME_RETRY")
			cmd.Flags().StringVarP(
				&opts.RegistryMirror, "registry-mirror", "m", "", "Registry mirror; or use GIVME_REGISTRY_MIRROR")
			cmd.Flags().StringVar(
				&opts.RegistryUsername, "registry-username", "", "Username for registry authentication; or use GIVME_REGISTRY_USERNAME")
			cmd.Flags().StringVar(
				&opts.RegistryPassword, "registry-password", "", "Password for registry authentication; or use GIVME_REGISTRY_PASSWORD")
		},
		saveCmd, exportCmd, getenvCmd, loadCmd, prootCmd,
	)

	// --entrypoint
	prootCmd.Flags().StringVar(
		&opts.Entrypoint, "entrypoint", "", "Entrypoint for the container")
}

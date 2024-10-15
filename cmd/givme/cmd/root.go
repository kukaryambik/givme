package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/logging"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	appName = "givme"
)

type CommandOptions struct {
	Cleanup          bool `mapstructure:"cleanup"`
	Cmd              []string
	DotenvFile       string   `mapstructure:"dotenv-file"`
	Eval             bool     `mapstructure:"eval"`
	IgnorePaths      []string `mapstructure:"ignore"`
	Image            string
	LogFormat        string   `mapstructure:"log-format"`
	LogLevel         string   `mapstructure:"log-level"`
	LogTimestamp     bool     `mapstructure:"log-timestamp"`
	ProotCwd         string   `mapstructure:"cwd"`
	ProotEntrypoint  string   `mapstructure:"entrypoint"`
	ProotFlags       []string `mapstructure:"proot-flags"`
	ProotVolumes     []string
	ProotUser        string `mapstructure:"user"`
	RegistryMirror   string `mapstructure:"registry-mirror"`
	RegistryPassword string `mapstructure:"registry-password"`
	RegistryUsername string `mapstructure:"registry-username"`
	Retry            int    `mapstructure:"retry"`
	RootFS           string `mapstructure:"rootfs"`
	TarFile          string `mapstructure:"tar-file"`
	Workdir          string `mapstructure:"workdir"`
}

// Command Options with default values
var opts = &CommandOptions{
	Cleanup:      true,
	Eval:         false,
	LogFormat:    logging.FormatColor,
	LogLevel:     logging.DefaultLevel,
	LogTimestamp: true,
	RootFS:       "/",
	Workdir:      filepath.Join(paths.GetExecDir(), "tmp"),
}

func mkFlags(c func(*cobra.Command), l ...*cobra.Command) {
	for _, cmd := range l {
		c(cmd)
	}
}

func init() {
	// Global flags
	// --rootfs -r
	RootCmd.PersistentFlags().StringVarP(
		&opts.RootFS, "rootfs", "r", opts.RootFS, "RootFS directory; or use GIVME_ROOTFS")
	RootCmd.MarkPersistentFlagDirname("rootfs")
	// --workdir
	RootCmd.PersistentFlags().StringVar(
		&opts.Workdir, "workdir", opts.Workdir, "Working directory; or use GIVME_WORKDIR")
	RootCmd.MarkPersistentFlagDirname("workdir")
	// --ignore -i
	RootCmd.PersistentFlags().StringSliceVarP(
		&opts.IgnorePaths, "ignore", "i", nil, "Ignore these paths; or use GIVME_IGNORE")

	// Logging flags
	// --verbosity -v
	RootCmd.PersistentFlags().StringVarP(
		&opts.LogLevel, "verbosity", "v", opts.LogLevel,
		"Log level (trace, debug, info, warn, error, fatal, panic)",
	)
	// --log-format
	RootCmd.PersistentFlags().StringVar(
		&opts.LogFormat, "log-format", opts.LogFormat,
		"Log format (text, color, json)",
	)
	// --log-timestamp
	RootCmd.PersistentFlags().BoolVar(
		&opts.LogTimestamp, "log-timestamp", opts.LogTimestamp,
		"Timestamp in log output",
	)

	// Subcommand flags
	// --eval -e
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().BoolVarP(&opts.Eval, "eval", "e", opts.Eval, "Output might be evaluated")
	},
		restoreCmd, loadCmd, getenvCmd,
	)
	// --tar-file -f
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().StringVarP(
			&opts.TarFile, "tar-file", "f", opts.TarFile, "Path to the tar file")
		cmd.MarkFlagFilename("tar-file", ".tar")
	},
		snapshotCmd, saveCmd, prootCmd, exportCmd, restoreCmd,
	)
	restoreCmd.MarkFlagRequired("tar-file")
	// --dotenv-file -d
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().StringVarP(
			&opts.DotenvFile, "dotenv-file", "d", opts.DotenvFile, "Path to the .env file")
		cmd.MarkFlagFilename("dotenv-file", ".env")
	},
		snapshotCmd, getenvCmd, restoreCmd,
	)
	// --retry and --registry-[mirror|username|password]
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().IntVar(
			&opts.Retry, "retry", 0, "Retry attempts of saving the image; or use GIVME_RETRY")
		cmd.Flags().StringVar(
			&opts.RegistryMirror, "registry-mirror", opts.RegistryMirror, "Registry mirror; or use GIVME_REGISTRY_MIRROR")
		cmd.Flags().StringVar(
			&opts.RegistryUsername, "registry-username", opts.RegistryUsername, "Username for registry authentication; or use GIVME_REGISTRY_USERNAME")
		cmd.Flags().StringVar(
			&opts.RegistryPassword, "registry-password", opts.RegistryPassword, "Password for registry authentication; or use GIVME_REGISTRY_PASSWORD")
	},
		saveCmd, exportCmd, getenvCmd, loadCmd, prootCmd,
	)
	// --entrypoint
	prootCmd.Flags().StringVar(
		&opts.ProotEntrypoint, "entrypoint", opts.ProotEntrypoint, "Entrypoint for the container")
	// --cleanup
	prootCmd.Flags().BoolVar(
		&opts.Cleanup, "cleanup", opts.Cleanup, "Clean up root directory before load")
	// --cwd -w
	prootCmd.Flags().StringVarP(
		&opts.ProotCwd, "cwd", "w", opts.ProotCwd, "Working directory for the container")
	// --volume
	prootCmd.Flags().StringArrayVar(
		&opts.ProotVolumes, "volume", opts.ProotVolumes, "Bind mount a volume")
	// --user -u
	prootCmd.Flags().StringVarP(
		&opts.ProotUser, "user", "u", opts.ProotUser, "User for the container")
	// --proot-flags
	prootCmd.Flags().StringSliceVar(
		&opts.ProotFlags, "proot-flags", opts.ProotFlags, "Additional flags for proot")
	prootCmd.Flags().MarkHidden("proot-flags")

	// Initialize viper and bind flags to environment variables
	viper.SetEnvPrefix(appName) // Environment variables prefixed with GIVME_
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // Automatically bind environment variables

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
		viper.BindPFlags(cmd.Flags())
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
		return snapshot(opts)
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore from a snapshot archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := restore(opts)
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
		return cleanup(opts)
	},
}

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save image to tar archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		_, err := save(opts)
		return err
	},
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export container image tar and config",
	Args:  cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		return export(opts)
	},
}

var getenvCmd = &cobra.Command{
	Use:   "getenv",
	Short: "Get container image environment variables",
	Args:  cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		err := getenv(opts)
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
		_, err := load(opts)
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
		return proot(opts)
	},
}

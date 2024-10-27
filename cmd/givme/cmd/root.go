package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kukaryambik/givme/pkg/logging"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	AppName = "givme"
)

var (
	Version   string
	Commit    string
	BuildDate string
)

type CommandOptions struct {
	Cleanup          bool `mapstructure:"cleanup"`
	Cmd              []string
	IgnorePaths      []string `mapstructure:"ignore"`
	Image            string
	LogFormat        string   `mapstructure:"log-format"`
	LogLevel         string   `mapstructure:"log-level"`
	LogTimestamp     bool     `mapstructure:"log-timestamp"`
	ProotCwd         string   `mapstructure:"cwd"`
	ProotEntrypoint  string   `mapstructure:"entrypoint"`
	ProotFlags       []string `mapstructure:"proot-flags"`
	ProotMounts      []string `mapstructure:"mount"`
	ProotUser        string   `mapstructure:"change-id"`
	RegistryMirror   string   `mapstructure:"registry-mirror"`
	RegistryPassword string   `mapstructure:"registry-password"`
	RegistryUsername string   `mapstructure:"registry-username"`
	Retry            int      `mapstructure:"retry"`
	RootFS           string   `mapstructure:"rootfs"`
	TarFile          string
	Workdir          string `mapstructure:"workdir"`
}

// Command Options with default values
var opts = &CommandOptions{
	Cleanup:      true,
	LogFormat:    logging.FormatColor,
	LogLevel:     logging.DefaultLevel,
	ProotUser:    "0:0",
	RootFS:       "/",
	Workdir:      filepath.Join(util.GetExecDir(), "tmp"),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func mkFlags(c func(*cobra.Command), l ...*cobra.Command) {
	for _, cmd := range l {
		c(cmd)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(
		&opts.RootFS, "rootfs", "r", opts.RootFS, "RootFS directory; or use GIVME_ROOTFS")
	rootCmd.MarkPersistentFlagDirname("rootfs")
	rootCmd.PersistentFlags().StringVar(
		&opts.Workdir, "workdir", opts.Workdir, "Working directory; or use GIVME_WORKDIR")
	rootCmd.MarkPersistentFlagDirname("workdir")
	rootCmd.PersistentFlags().StringSliceVarP(
		&opts.IgnorePaths, "ignore", "i", nil, "Ignore these paths; or use GIVME_IGNORE")

	// Logging flags
	rootCmd.PersistentFlags().StringVarP(
		&opts.LogLevel, "verbosity", "v", opts.LogLevel, "Log level (trace, debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().StringVar(
		&opts.LogFormat, "log-format", opts.LogFormat, "Log format (text, color, json)")
	rootCmd.PersistentFlags().BoolVar(
		&opts.LogTimestamp, "log-timestamp", opts.LogTimestamp, "Timestamp in log output")

	// Subcommand flags
	// --tar-file -f
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().StringVarP(
			&opts.TarFile, "tar-file", "f", opts.TarFile, "Path to the tar file")
		cmd.MarkFlagFilename("tar-file", ".tar")
	},
		// Add them to the list of subcommands
		snapshotCmd, saveCmd,
	)
	// --retry and --registry-[mirror|username|password]
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().IntVar(
			&opts.Retry, "retry", 0, "Retry attempts of downloading the image; or use GIVME_RETRY")
		cmd.Flags().StringVar(
			&opts.RegistryMirror, "registry-mirror", opts.RegistryMirror, "Registry mirror; or use GIVME_REGISTRY_MIRROR")
		cmd.Flags().StringVar(
			&opts.RegistryUsername, "registry-username", opts.RegistryUsername, "Username for registry authentication; or use GIVME_REGISTRY_USERAppName")
		cmd.Flags().StringVar(
			&opts.RegistryPassword, "registry-password", opts.RegistryPassword, "Password for registry authentication; or use GIVME_REGISTRY_PASSWORD")
	},
		// Add them to the list of subcommands
		saveCmd, loadCmd, runCmd,
	)

	// --cleanup
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().BoolVar(
			&opts.Cleanup, "cleanup", opts.Cleanup, "Clean up root directory before load")
	},
		// Add them to the list of subcommands
		loadCmd, runCmd,
	)

	runCmd.Flags().StringVar(
		&opts.ProotEntrypoint, "entrypoint", opts.ProotEntrypoint, "Entrypoint for the container")
	runCmd.Flags().StringVarP(
		&opts.ProotCwd, "cwd", "w", opts.ProotCwd, "Working directory for the container")
	runCmd.Flags().StringSliceVar(
		&opts.ProotMounts, "mount", opts.ProotMounts, "Mount host path to the container")
	runCmd.Flags().StringVarP(
		&opts.ProotUser, "change-id", "u", opts.ProotUser, "UID:GID for the container")
	runCmd.Flags().StringSliceVar(
		&opts.ProotFlags, "proot-flags", opts.ProotFlags, "Additional flags for proot")
	runCmd.Flags().MarkHidden("proot-flags")

	// Initialize viper and bind flags to environment variables
	viper.SetEnvPrefix(AppName) // Environment variables prefixed with GIVME_
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // Automatically bind environment variables

	// Add subcommands for snapshot, restore, and cleanup.
	rootCmd.AddCommand(
		cleanupCmd,
		loadCmd,
		runCmd,
		saveCmd,
		snapshotCmd,
		versionCmd,
	)
}

var rootCmd = &cobra.Command{
	Use:   AppName,
	Short: fmt.Sprintf("%s - Switch the image from inside the container", AppName),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set variables from flags or environment
		viper.BindPFlags(cmd.Flags())
		viper.Unmarshal(&opts)

		// Set up logging
		if err := logging.Configure(opts.LogLevel, opts.LogFormat, opts.LogTimestamp, true); err != nil {
			return err
		}

		// Check if rootfs and workdir are the same
		if opts.RootFS == opts.Workdir {
			return fmt.Errorf("rootfs and workdir cannot be the same")
		}

		// Ensure the work directory exists.
		if err := os.MkdirAll(opts.Workdir, os.ModePerm); err != nil {
			logrus.Fatalf("Error creating work directory: %v", err)
		}

		return nil
	},
}

var snapshotCmd = &cobra.Command{
	Use:     "snapshot",
	Aliases: []string{"snap"},
	Short:   "Create a snapshot archive",
	PreRun: func(cmd *cobra.Command, args []string) {
		if opts.TarFile == "" {
			opts.TarFile = filepath.Join(opts.Workdir, defaultSnapshotFile)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return snapshot(opts)
	},
}

var cleanupCmd = &cobra.Command{
	Use:     "cleanup",
	Aliases: []string{"c", "clean"},
	Short:   "Clean up directories",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return cleanup(opts)
	},
}

var saveCmd = &cobra.Command{
	Use:     "save [flags] IMAGE",
	Aliases: []string{"download", "pull"},
	Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	Short:   "Save image to tar archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		cmd.SilenceUsage = true
		_, err := save(opts)
		return err
	},
}

var loadCmd = &cobra.Command{
	Use:     "load [flags] IMAGE",
	Aliases: []string{"l", "lo", "loa"},
	Example: fmt.Sprintf("source <(%s load alpine)", AppName),
	Short:   "Extract the container filesystem to the rootfs directory",
	Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		cmd.SilenceUsage = true
		_, err := load(opts)
		if err != nil {
			fmt.Print("false")
		}
		return err
	},
}

var runCmd = &cobra.Command{
	Use:     "run [flags] IMAGE [cmd]...",
	Aliases: []string{"r", "proot"},
	Short:   "Run a command in the container",
	Args:    cobra.MinimumNArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		opts.Cmd = args[1:]
		cmd.SilenceUsage = true
		return run(opts)
	},
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v", "ver"},
	Short:   "Display version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf(
			"Version: %s\nCommit: %s\nBuild Date: %s\nPlatform: %s\n",
			Version, Commit, BuildDate, runtime.GOOS+"/"+runtime.GOARCH,
		)
	},
}

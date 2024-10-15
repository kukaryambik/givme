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
	NAME    = "givme"
	VERSION = "0.0.0"
)

type CommandOptions struct {
	Cleanup          bool `mapstructure:"cleanup"`
	Cmd              []string
	DotenvFile       string
	Eval             bool     `mapstructure:"eval"`
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
	Eval:         false,
	LogFormat:    logging.FormatColor,
	LogLevel:     logging.DefaultLevel,
	LogTimestamp: true,
	ProotUser:    "0:0",
	RootFS:       "/",
	Workdir:      filepath.Join(paths.GetExecDir(), "tmp"),
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
	// --eval -E
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().BoolVarP(&opts.Eval, "eval", "E", opts.Eval, "Output might be evaluated")
	},
		// Add them to the list of subcommands
		restoreCmd, loadCmd, getenvCmd,
	)
	// --tar-file -f
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().StringVarP(
			&opts.TarFile, "tar-file", "f", opts.TarFile, "Path to the tar file")
		cmd.MarkFlagFilename("tar-file", ".tar")
	},
		// Add them to the list of subcommands
		snapshotCmd, saveCmd, exportCmd,
	)
	// --dotenv-file -d
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().StringVarP(
			&opts.DotenvFile, "dotenv-file", "d", opts.DotenvFile, "Path to the .env file")
		cmd.MarkFlagFilename("dotenv-file", ".env")
	},
		// Add them to the list of subcommands
		snapshotCmd, getenvCmd, restoreCmd,
	)
	// --retry and --registry-[mirror|username|password]
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().IntVar(
			&opts.Retry, "retry", 0, "Retry attempts of downloading the image; or use GIVME_RETRY")
		cmd.Flags().StringVar(
			&opts.RegistryMirror, "registry-mirror", opts.RegistryMirror, "Registry mirror; or use GIVME_REGISTRY_MIRROR")
		cmd.Flags().StringVar(
			&opts.RegistryUsername, "registry-username", opts.RegistryUsername, "Username for registry authentication; or use GIVME_REGISTRY_USERNAME")
		cmd.Flags().StringVar(
			&opts.RegistryPassword, "registry-password", opts.RegistryPassword, "Password for registry authentication; or use GIVME_REGISTRY_PASSWORD")
	},
		// Add them to the list of subcommands
		saveCmd, exportCmd, getenvCmd, loadCmd, runCmd,
	)

	runCmd.Flags().StringVar(
		&opts.ProotEntrypoint, "entrypoint", opts.ProotEntrypoint, "Entrypoint for the container")
	runCmd.Flags().BoolVar(
		&opts.Cleanup, "cleanup", opts.Cleanup, "Clean up root directory before load")
	runCmd.Flags().StringVarP(
		&opts.ProotCwd, "cwd", "w", opts.ProotCwd, "Working directory for the container")
	runCmd.Flags().StringArrayVar(
		&opts.ProotMounts, "mount", opts.ProotMounts, "Mount host path to the container")
	runCmd.Flags().StringVarP(
		&opts.ProotUser, "change-id", "u", opts.ProotUser, "UID:GID for the container")
	runCmd.Flags().StringSliceVar(
		&opts.ProotFlags, "proot-flags", opts.ProotFlags, "Additional flags for proot")
	runCmd.Flags().MarkHidden("proot-flags")

	// Initialize viper and bind flags to environment variables
	viper.SetEnvPrefix(NAME) // Environment variables prefixed with GIVME_
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // Automatically bind environment variables

	// Add subcommands for snapshot, restore, and cleanup.
	rootCmd.AddCommand(
		cleanupCmd,
		exportCmd,
		getenvCmd,
		loadCmd,
		runCmd,
		restoreCmd,
		saveCmd,
		snapshotCmd,
		versionCmd,
	)
}

var rootCmd = &cobra.Command{
	Use:   NAME,
	Short: fmt.Sprintf("%s - Switch the image from inside the container", NAME),
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
			opts.TarFile = filepath.Join(opts.Workdir, defaultSnapshotName+".tar")
		}
		if opts.DotenvFile == "" {
			opts.DotenvFile = filepath.Join(opts.Workdir, defaultSnapshotName+".env")
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return snapshot(opts)
	},
}

var restoreCmd = &cobra.Command{
	Use:     "restore [flags] FILE",
	Aliases: []string{"rstr"},
	Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	Short:   "Restore from a snapshot archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.TarFile = args[0]
		cmd.SilenceUsage = true
		err := restore(opts)
		if err != nil && opts.Eval {
			fmt.Print("false")
		}
		return err
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

var exportCmd = &cobra.Command{
	Use:     "export [flags] IMAGE",
	Aliases: []string{"e"},
	Short:   "Export container filesystem as a tarball",
	Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		cmd.SilenceUsage = true
		return export(opts)
	},
}

var getenvCmd = &cobra.Command{
	Use:     "getenv [flags] IMAGE",
	Aliases: []string{"env"},
	Short:   "Get container image environment variables",
	Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		cmd.SilenceUsage = true
		err := getenv(opts)
		if err != nil && opts.Eval {
			fmt.Print("false")
		}
		return err
	},
}

var loadCmd = &cobra.Command{
	Use:     "load [flags] IMAGE",
	Aliases: []string{"l"},
	Short:   "Extract the container filesystem to the rootfs directory",
	Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		cmd.SilenceUsage = true
		_, err := load(opts)
		if err != nil && opts.Eval {
			fmt.Print("false")
		}
		return err
	},
}

var runCmd = &cobra.Command{
	Use:     "run [flags] IMAGE [cmd]...",
	Aliases: []string{"r", "proot"},
	Short:   "Run a command in a container",
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
		fmt.Println("Version: ", VERSION)
	},
}

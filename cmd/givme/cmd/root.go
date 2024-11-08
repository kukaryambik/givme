package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kukaryambik/givme/pkg/logging"
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
	Cmd              []string
	Cwd              string
	Entrypoint       []string
	IgnorePaths      []string `mapstructure:"ignore"`
	Image            string
	LogFormat        string `mapstructure:"log-format"`
	LogLevel         string `mapstructure:"log-level"`
	LogTimestamp     bool   `mapstructure:"log-timestamp"`
	NoPurge          bool
	OverwriteEnv     bool
	RegistryMirror   string `mapstructure:"registry-mirror"`
	RegistryPassword string `mapstructure:"registry-password"`
	RegistryUsername string `mapstructure:"registry-username"`
	RootFS           string `mapstructure:"rootfs"`
	RunChangeID      string
	RunName          string
	RunProotBinds    []string `mapstructure:"proot-bind"`
	RunProotBin      string   `mapstructure:"proot-bin"`
	RunProotFlags    string   `mapstructure:"proot-flags"`
	RunRemoveAfter   bool
	TarFile          string
	Update           bool   `mapstructure:"update"`
	Workdir          string `mapstructure:"workdir"`
}

// Command Options with default values
var opts = &CommandOptions{
	LogFormat: logging.FormatColor,
	LogLevel:  logging.DefaultLevel,
	RootFS:    "/",
	Workdir:   filepath.Join("/tmp", AppName),
}

var (
	defaultImagesDir  = func() string { return filepath.Join(opts.Workdir, "images") }
	defaultLayersDir  = func() string { return filepath.Join(opts.Workdir, "layers") }
	defaultCacheDir   = func() string { return filepath.Join(opts.Workdir, "cache") }
	defaultDotEnvFile = func() string { return filepath.Join(opts.Workdir, "last.env") }
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func init() {
	a := strings.ToUpper(AppName)

	// Global flags
	rootCmd.PersistentFlags().StringVarP(
		&opts.RootFS, "rootfs", "r", opts.RootFS, fmt.Sprintf("RootFS directory; or use %s_ROOTFS", a))
	rootCmd.MarkPersistentFlagDirname("rootfs")
	rootCmd.PersistentFlags().StringVar(
		&opts.Workdir, "workdir", opts.Workdir, fmt.Sprintf("Working directory; or use %s_WORKDIR", a))
	rootCmd.MarkPersistentFlagDirname("workdir")
	rootCmd.PersistentFlags().StringSliceVarP(
		&opts.IgnorePaths, "ignore", "i", nil, fmt.Sprintf("Ignore these paths; or use %s_IGNORE", a))
	rootCmd.Flags().StringVar(
		&opts.RegistryMirror, "registry-mirror", opts.RegistryMirror,
		fmt.Sprintf("Registry mirror; or use %s_REGISTRY_MIRROR", strings.ToUpper(AppName)),
	)
	rootCmd.Flags().StringVar(
		&opts.RegistryUsername, "registry-username", opts.RegistryUsername,
		fmt.Sprintf("Username for registry authentication; or use %s_REGISTRY_USERNAME", a),
	)
	rootCmd.Flags().StringVar(
		&opts.RegistryPassword, "registry-password", opts.RegistryPassword,
		fmt.Sprintf("Password for registry authentication; or use %s_REGISTRY_PASSWORD", a),
	)

	// Logging flags
	rootCmd.PersistentFlags().StringVarP(
		&opts.LogLevel, "verbosity", "v", opts.LogLevel, "Log level (trace, debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().StringVar(
		&opts.LogFormat, "log-format", opts.LogFormat, "Log format (text, color, json)")
	rootCmd.PersistentFlags().BoolVar(
		&opts.LogTimestamp, "log-timestamp", opts.LogTimestamp, "Timestamp in log output")

	// Add subcommands
	rootCmd.AddCommand(
		ApplyCmd(),
		ExecCmd(),
		extractCmd(),
		getenvCmd(),
		PurgeCmd(),
		RunCmd(),
		SaveCmd(),
		SnapshotCmd(),
		versionCmd,
	)

	// Initialize viper and bind flags to environment variables
	viper.SetEnvPrefix(AppName) // Environment variables prefixed with GIVME_
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // Automatically bind environment variables
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

		// Create default directories
		for _, p := range []string{defaultImagesDir(), defaultLayersDir(), defaultCacheDir()} {
			if err := os.MkdirAll(p, os.ModePerm); err != nil {
				logrus.Fatalf("Error creating directory %s: %v", p, err)
			}
		}

		return nil
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

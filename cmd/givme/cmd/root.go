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
	IgnorePaths      []string `mapstructure:"ignore"`
	Image            string
	LogFormat        string   `mapstructure:"log-format"`
	LogLevel         string   `mapstructure:"log-level"`
	LogTimestamp     bool     `mapstructure:"log-timestamp"`
	NoPurge          bool     `mapstructure:"no-purge"`
	OverwriteEnv     bool     `mapstructure:"overwrite-env"`
	Cwd              string   `mapstructure:"cwd"`
	ProotFlags       []string `mapstructure:"proot-flags"`
	ProotMounts      []string `mapstructure:"proot-mount"`
	ChangeID         string   `mapstructure:"change-id"`
	RedirectOutput   bool
	RegistryMirror   string `mapstructure:"registry-mirror"`
	RegistryPassword string `mapstructure:"registry-password"`
	RegistryUsername string `mapstructure:"registry-username"`
	RootFS           string `mapstructure:"rootfs"`
	Shell            string `mapstructure:"shell"`
	TarFile          string
	Update           bool   `mapstructure:"update"`
	Workdir          string `mapstructure:"workdir"`
}

// Command Options with default values
var opts = &CommandOptions{
	LogFormat: logging.FormatColor,
	LogLevel:  logging.DefaultLevel,
	ChangeID:  "0:0",
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

func mkFlags(c func(*cobra.Command), l ...*cobra.Command) {
	for _, cmd := range l {
		c(cmd)
	}
}

func init() {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		logrus.Warnf("Failed to stat Stdout: %v", err)
	}

	if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		opts.RedirectOutput = true
		logrus.Debug("Stdout is not a terminal")
	}

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
	// --registry-[mirror|username|password]
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().StringVar(
			&opts.RegistryMirror, "registry-mirror", opts.RegistryMirror,
			fmt.Sprintf("Registry mirror; or use %s_REGISTRY_MIRROR", strings.ToUpper(AppName)),
		)
		cmd.Flags().StringVar(
			&opts.RegistryUsername, "registry-username", opts.RegistryUsername,
			fmt.Sprintf("Username for registry authentication; or use %s_REGISTRY_USERNAME", a),
		)
		cmd.Flags().StringVar(
			&opts.RegistryPassword, "registry-password", opts.RegistryPassword,
			fmt.Sprintf("Password for registry authentication; or use %s_REGISTRY_PASSWORD", a),
		)
	},
		// Add them to the list of subcommands
		saveCmd, applyCmd, runCmd, getenvCmd,
	)
	// --update
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().BoolVar(
			&opts.Update, "update", opts.Update, "Update the image instead of using existing file")
	},
		// Add them to the list of subcommands
		applyCmd, runCmd,
	)
	// --no-purge
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().BoolVar(
			&opts.NoPurge, "no-purge", opts.NoPurge, "Do not purge the root directory before unpacking the image")
	},
		// Add them to the list of subcommands
		applyCmd, runCmd,
	)
	// --overwrite-env
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().BoolVar(
			&opts.OverwriteEnv, "overwrite-env", opts.OverwriteEnv, "Overwrite current environment variables with new ones from the image")
	},
		// Add them to the list of subcommands
		applyCmd, runCmd,
	)
	// --shell
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().StringVar(
			&opts.Shell, "shell", opts.Shell, "Shell to use for the container")
	},
		// Add them to the list of subcommands
		applyCmd, runCmd,
	)

	runCmd.Flags().StringVarP(
		&opts.Cwd, "cwd", "w", opts.Cwd, "Working directory for the container")
	runCmd.Flags().StringSliceVar(
		&opts.ProotMounts, "mount", opts.ProotMounts, "Mount host path to the container")
	runCmd.Flags().StringVarP(
		&opts.ChangeID, "change-id", "u", opts.ChangeID, "UID:GID for the container")
	runCmd.Flags().StringSliceVar(
		&opts.ProotFlags, "proot-flags", opts.ProotFlags, "Additional flags for proot")
	runCmd.Flags().MarkHidden("proot-flags")

	// Initialize viper and bind flags to environment variables
	viper.SetEnvPrefix(AppName) // Environment variables prefixed with GIVME_
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // Automatically bind environment variables

	// Add subcommands
	rootCmd.AddCommand(
		purgeCmd,
		applyCmd,
		runCmd,
		getenvCmd,
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

		// Create default directories
		for _, p := range []string{defaultImagesDir(), defaultLayersDir(), defaultCacheDir()} {
			if err := os.MkdirAll(p, os.ModePerm); err != nil {
				logrus.Fatalf("Error creating directory %s: %v", p, err)
			}
		}

		return nil
	},
}

var snapshotCmd = &cobra.Command{
	Use:     "snapshot",
	Aliases: []string{"snap"},
	Short:   "Create a snapshot archive",
	Example: fmt.Sprintf("SNAPSHOT=$(%s snap)", AppName),
	PreRun: func(cmd *cobra.Command, args []string) {
		if opts.TarFile == "" {
			opts.TarFile = filepath.Join(defaultImagesDir(), defaultSnapshotFile())
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return opts.snapshot()
	},
}

var purgeCmd = &cobra.Command{
	Use:     "purge",
	Aliases: []string{"p", "clear"},
	Short:   "Purge the rootfs directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return opts.purge()
	},
}

var saveCmd = &cobra.Command{
	Use:     "save [flags] IMAGE",
	Aliases: []string{"download", "pull"},
	Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	Short:   "Save image to tar archive",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		opts.Update = true
		cmd.SilenceUsage = true
		img, err := opts.save()
		fmt.Println(img.File)
		return err
	},
}

var getenvCmd = &cobra.Command{
	Use:     "getenv [flags] IMAGE",
	Aliases: []string{"e", "env"},
	Short:   "Get environment variables from image",
	Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		cmd.SilenceUsage = true
		return opts.getenv()
	},
}

var applyCmd = &cobra.Command{
	Use:     "apply [flags] IMAGE [cmd]...",
	Aliases: []string{"a", "an", "the", "load", "l"},
	Example: fmt.Sprintf("exec %s apply alpine", AppName),
	Short:   "Extract the container filesystem to the rootfs directory and run a command",
	Args:    cobra.MinimumNArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		opts.Cmd = args[1:]
		cmd.SilenceUsage = true
		_, err := opts.apply()
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
		return opts.run()
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

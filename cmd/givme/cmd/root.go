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
	"github.com/spf13/pflag"
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

func AddFlag(c func(*pflag.FlagSet), l ...*cobra.Command) {
	for _, cmd := range l {
		c(cmd.Flags())
	}
}

func mkFlags(c func(*cobra.Command), l ...*cobra.Command) {
	for _, cmd := range l {
		c(cmd)
		cmd.Flags()
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
		&opts.LogLevel, "verbosity", "v", "", "Log level (trace, debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().StringVar(
		&opts.LogFormat, "log-format", opts.LogFormat, "Log format (text, color, json)")
	rootCmd.PersistentFlags().BoolVar(
		&opts.LogTimestamp, "log-timestamp", opts.LogTimestamp, "Timestamp in log output")

	var cmd *cobra.Command
	// Subcommand flags
	cmd.Flags().StringVarP(&opts.TarFile, "tar-file", "f", "", "Path to the tar file")
	cmd.MarkFlagFilename("tar-file", ".tar")
	// saveCmd,

	// --update
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().BoolVar(
			&opts.Update, "update", opts.Update, "Update the image instead of using existing file")
	},
		// Add them to the list of subcommands
		runCmd, extractCmd,
	)

	// --overwrite-env
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().BoolVar(
			&opts.OverwriteEnv, "overwrite-env", opts.OverwriteEnv, "Overwrite current environment variables with new ones from the image")
	},
		// Add them to the list of subcommands
		runCmd,
	)

	// --entrypoint, --cwd
	mkFlags(func(cmd *cobra.Command) {
		cmd.Flags().StringArrayVar(
			&opts.Entrypoint, "entrypoint", opts.Entrypoint, "Entrypoint for the container")
		cmd.Flags().StringVarP(
			&opts.Cwd, "cwd", "w", opts.Cwd, "Working directory for the container")
	},
		// Add them to the list of subcommands
		runCmd,
	)

	runCmd.Flags().StringVarP(&opts.RunChangeID, "change-id", "u", opts.RunChangeID, "UID:GID for the container")
	runCmd.Flags().StringArrayVarP(
		&opts.RunProotBinds, "proot-bind", "b", opts.RunProotBinds, "Mount host path to the container")
	rootCmd.Flags().AddFlag(cmd.Flag("proot-bind"))

	runCmd.Flags().BoolVar(
		&opts.RunRemoveAfter, "rm", opts.RunRemoveAfter, "Remove the rootfs directory after running the command")
	runCmd.Flags().StringVar(
		&opts.RunName, "name", opts.RunName, "The name of the container")
	runCmd.Flags().StringVar(
		&opts.RunProotFlags, "proot-flags", opts.RunProotFlags, "Additional flags for proot")
	runCmd.Flags().MarkHidden("proot-flags")
	runCmd.Flags().StringVar(
		&opts.RunProotBin, "proot-bin", opts.RunProotBin, "Path to the proot binary")

	// Initialize viper and bind flags to environment variables
	viper.SetEnvPrefix(AppName) // Environment variables prefixed with GIVME_
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // Automatically bind environment variables

	// Add subcommands
	rootCmd.AddCommand(
		ApplyCmd(),
		ExecCmd(),
		extractCmd,
		getenvCmd,
		purgeCmd,
		runCmd,
		saveCmd,
		SnapshotCmd(),
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

var purgeCmd = &cobra.Command{
	Use:     "purge",
	Aliases: []string{"p", "clear"},
	Short:   "Purge the rootfs directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return opts.Purge()
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
		img, err := opts.Save()
		fmt.Println(img.File)
		return err
	},
}

var getenvCmd = &cobra.Command{
	Use:     "getenv [flags] IMAGE",
	Aliases: []string{"env"},
	Short:   "Get environment variables from image",
	Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		cmd.SilenceUsage = true
		return opts.Getenv()
	},
}

var extractCmd = &cobra.Command{
	Use:     "extract [flags] IMAGE",
	Aliases: []string{"ex", "ext", "unpack"},
	Short:   "Extract the image filesystem",
	Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	RunE: func(cmd *cobra.Command, args []string) error {
		opts.Image = args[0]
		cmd.SilenceUsage = true
		_, err := opts.Extract()
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
		return opts.Run()
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

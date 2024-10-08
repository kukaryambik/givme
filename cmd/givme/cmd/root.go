package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/exclusions"
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
	UserExclusions   string
	Workdir          string
	Eval             bool
}

var (
	logLevel     string
	logFormat    string
	logTimestamp bool

	cleanupConf  CommandOptions
	exportConf   CommandOptions
	loadConf     CommandOptions
	restoreConf  CommandOptions
	rootConf     CommandOptions
	saveConf     CommandOptions
	snapshotConf CommandOptions
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
		rootConf.Workdir = viper.GetString("workdir")
		rootConf.RootFS = viper.GetString("rootfs")
		rootConf.UserExclusions = viper.GetString("exclude")
		rootConf.RegistryMirror = viper.GetString("registry-mirror")
		rootConf.RegistryUsername = viper.GetString("registry-username")
		rootConf.RegistryPassword = viper.GetString("registry-password")
		rootConf.Retry = viper.GetInt("retry")
		logLevel = viper.GetString("verbosity")

		// Set up logging
		if err := logging.Configure(logLevel, logFormat, logTimestamp, rootConf.Eval); err != nil {
			return err
		}

		// Ensure the work directory exists.
		if err := os.MkdirAll(rootConf.Workdir, 0755); err != nil {
			logrus.Fatalf("Error creating work directory: %v", err)
		}

		// Build exclusions
		if excl, err := exclusions.Build(rootConf.UserExclusions, rootConf.Workdir); err != nil {
			return err
		} else {
			rootConf.Exclusions = excl
		}
		return nil
	},
}

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Create a snapshot archive",
	PreRun: func(cmd *cobra.Command, args []string) {
		rootConf.TarFile = filepath.Join(rootConf.Workdir, defaultSnapshotName+".tar")
		rootConf.DotenvFile = filepath.Join(rootConf.Workdir, defaultSnapshotName+".env")
		util.MergeStructs(&rootConf, &snapshotConf)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return snapshot(&snapshotConf)
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore from a snapshot archive",
	PreRun: func(cmd *cobra.Command, args []string) {
		util.MergeStructs(&rootConf, &restoreConf)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := restore(&restoreConf)
		if err != nil && restoreConf.Eval {
			fmt.Print("false")
		}
		return err
	},
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up directories",
	PreRun: func(cmd *cobra.Command, args []string) {
		util.MergeStructs(&rootConf, &cleanupConf)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cleanup(&cleanupConf)
	},
}

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save image to tar archive",
	PreRun: func(cmd *cobra.Command, args []string) {
		rootConf.Image = args[0]
		imgSlug := util.Slugify(rootConf.Image)
		rootConf.TarFile = filepath.Join(rootConf.Workdir, imgSlug+".tar")
		util.MergeStructs(&rootConf, &saveConf)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := save(&saveConf)
		return err
	},
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export container image tar and config",
	Args:  cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	PreRun: func(cmd *cobra.Command, args []string) {
		rootConf.Image = args[0]
		imgSlug := util.Slugify(rootConf.Image)
		rootConf.TarFile = filepath.Join(rootConf.Workdir, imgSlug+".tar")
		rootConf.DotenvFile = filepath.Join(rootConf.Workdir, imgSlug+".env")

		util.MergeStructs(&rootConf, &exportConf)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return export(&exportConf)
	},
}

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Load container image tar and apply it to the system",
	Args:  cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	PreRun: func(cmd *cobra.Command, args []string) {
		rootConf.Image = args[0]
		imgSlug := util.Slugify(rootConf.Image)
		rootConf.TarFile = filepath.Join(rootConf.Workdir, imgSlug+".tar")
		util.MergeStructs(&rootConf, &loadConf)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := load(&loadConf)
		if err != nil && loadConf.Eval {
			fmt.Print("false")
		}
		return err
	},
}

func addFlags() {
	execdir := util.GetExecDir()

	// RootCmd flags
	RootCmd.PersistentFlags().StringVar(
		&rootConf.RootFS, "rootfs", "/", "RootFS directory; or use GIVME_ROOTFS")
	RootCmd.PersistentFlags().StringVar(
		&rootConf.Workdir, "workdir", execdir, "Working directory; or use GIVME_WORKDIR")
	RootCmd.PersistentFlags().StringVar(
		&rootConf.UserExclusions, "exclude", "", "Excluded directories; or use GIVME_EXCLUDE")
	RootCmd.PersistentFlags().BoolVarP(
		&rootConf.Eval, "eval", "e", false, "Output might be evaluated")
	RootCmd.PersistentFlags().StringVarP(
		&rootConf.TarFile, "tar-file", "f", "", "Path to the tar file")
	RootCmd.PersistentFlags().IntVar(
		&rootConf.Retry, "retry", 0, "Retry attempts of saving the image; or use GIVME_RETRY")
	RootCmd.PersistentFlags().StringVar(
		&rootConf.RegistryMirror, "registry-mirror", "",
		"Registry mirror; or use GIVME_REGISTRY_MIRROR",
	)
	RootCmd.PersistentFlags().StringVar(
		&exportConf.RegistryUsername, "registry-username", "",
		"Username for registry authentication; or use GIVME_REGISTRY_USERNAME",
	)
	RootCmd.PersistentFlags().StringVar(
		&exportConf.RegistryPassword, "registry-password", "",
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

	// snapshotCmd flags
	snapshotCmd.Flags().StringVarP(
		&snapshotConf.DotenvFile, "dotenv-file", "d", "", "Path to the .env file")

	// restoreCmd flags
	restoreCmd.MarkFlagRequired("tar-file")
	restoreCmd.Flags().StringVarP(
		&restoreConf.DotenvFile, "dotenv-file", "d", "", "Path to the .env file")

	// exportCmd flags
	exportCmd.Flags().StringVarP(
		&exportConf.DotenvFile, "dotenv-file", "d", "", "Path to the .env file")
}

package cmd

import (
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
	ConfigFile       string
	DotenvFile       string
	Exclusions       []string
	Image            string
	RegistryUsername string
	RegistryPassword string
	RootFS           string
	TarFile          string
	UserExclusions   string
	Workdir          string
}

var (
	logLevel     string
	logFormat    string
	logTimestamp bool

	rootConf     CommandOptions
	snapshotConf CommandOptions
	restoreConf  CommandOptions
	cleanupConf  CommandOptions
	exportConf   CommandOptions
	loadConf     CommandOptions
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
		snapshotCmd,
		restoreCmd,
		exportCmd,
		loadCmd,
	)
}

var RootCmd = &cobra.Command{
	Use: appName,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set variables from flags or environment
		rootConf.Workdir = viper.GetString("workdir")
		rootConf.RootFS = viper.GetString("rootfs")
		rootConf.UserExclusions = viper.GetString("exclude")
		logLevel = viper.GetString("verbosity")

		// Set up logging
		if err := logging.Configure(logLevel, logFormat, logTimestamp); err != nil {
			return err
		}

		logrus.Debugf("Config: %s", rootConf)

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
		util.MergeStructs(&rootConf, &snapshotConf)
		snapshotConf.TarFile = filepath.Join(snapshotConf.Workdir, "snapshot.tar")
		snapshotConf.DotenvFile = filepath.Join(snapshotConf.Workdir, ".env")
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
		return restore(&restoreConf)
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

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export container image tar and config",
	Args:  cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	PreRun: func(cmd *cobra.Command, args []string) {
		util.MergeStructs(&rootConf, &exportConf)
		exportConf.Image = args[0]
		imgSlug := util.Slugify(exportConf.Image)
		exportConf.TarFile = filepath.Join(exportConf.Workdir, imgSlug+".tar")
		exportConf.ConfigFile = filepath.Join(exportConf.Workdir, imgSlug+".json")
		exportConf.DotenvFile = filepath.Join(exportConf.Workdir, imgSlug+".env")

		viper.BindPFlags(cmd.Flags())
		exportConf.RegistryUsername = viper.GetString("registry-username")
		exportConf.RegistryPassword = viper.GetString("registry-password")
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := export(&exportConf)
		return err
	},
}

var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "Load container image tar and apply it to the system",
	Args:  cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
	PreRun: func(cmd *cobra.Command, args []string) {
		util.MergeStructs(&rootConf, &loadConf)
		loadConf.Image = args[0]
		loadConf.TarFile = filepath.Join(loadConf.Workdir, "snapshot.tar")
		loadConf.DotenvFile = filepath.Join(loadConf.Workdir, ".env")

		viper.BindPFlags(cmd.Flags())
		loadConf.RegistryUsername = viper.GetString("registry-username")
		loadConf.RegistryPassword = viper.GetString("registry-password")
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return load(&loadConf)
	},
}

func addFlags() {
	execdir := util.GetExecDir()

	// RootCmd flags
	RootCmd.PersistentFlags().StringVar(
		&rootConf.RootFS, "rootfs", "/", "RootFS directory")
	RootCmd.PersistentFlags().StringVar(
		&rootConf.Workdir, "workdir", execdir, "Working directory")
	RootCmd.PersistentFlags().StringVar(
		&rootConf.UserExclusions, "exclude", "", "Excluded directories")
	// Logging flags
	RootCmd.PersistentFlags().StringVarP(
		&logLevel, "verbosity", "v", logging.DefaultLevel, "Log level (trace, debug, info, warn, error, fatal, panic)")
	RootCmd.PersistentFlags().StringVar(
		&logFormat, "log-format", logging.FormatColor, "Log format (text, color, json)")
	RootCmd.PersistentFlags().BoolVar(
		&logTimestamp, "log-timestamp", logging.DefaultLogTimestamp, "Timestamp in log output")

	// snapshotCmd flags
	snapshotCmd.Flags().StringVarP(
		&snapshotConf.TarFile, "tar-file", "f", "", "Path to the snapshot archive file")
	snapshotCmd.Flags().StringVarP(
		&snapshotConf.DotenvFile, "dotenv-file", "e", "", "Path to the .env file")

	// restoreCmd flags
	restoreCmd.Flags().StringVarP(
		&restoreConf.TarFile, "tar-file", "f", "", "Path to the snapshot archive file")
	restoreCmd.MarkFlagRequired("tar-file")
	restoreCmd.Flags().StringVarP(
		&restoreConf.DotenvFile, "dotenv-file", "e", "", "Path to the .env file")

	// exportCmd flags
	exportCmd.Flags().StringVarP(
		&exportConf.TarFile, "tar-file", "f", "", "Path to the tar file")
	exportCmd.Flags().StringVarP(
		&exportConf.ConfigFile, "config-file", "c", "", "Path to the config file")
	exportCmd.Flags().StringVarP(
		&exportConf.DotenvFile, "dotenv-file", "e", "", "Path to the .env file")
	exportCmd.Flags().StringVar(
		&exportConf.RegistryUsername, "registry-username", "", "Username for registry authentication")
	exportCmd.Flags().StringVar(
		&exportConf.RegistryPassword, "registry-password", "", "Password for registry authentication")

	// loadCmd flags
	loadCmd.Flags().StringVarP(
		&loadConf.TarFile, "tar-file", "f", "", "Path to the snapshot archive file")
	loadCmd.Flags().StringVarP(
		&loadConf.DotenvFile, "dotenv-file", "e", "", "Path to the .env file")
	loadCmd.Flags().StringVar(
		&loadConf.RegistryUsername, "registry-username", "", "Username for registry authentication")
	loadCmd.Flags().StringVar(
		&loadConf.RegistryPassword, "registry-password", "", "Password for registry authentication")
}
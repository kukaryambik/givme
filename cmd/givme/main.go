package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const appName = "givme"

var (
	dotenv, file, workDir, rootfs, exclude, verbose string
)

// getLogger initializes and returns a logger
func getLogger(v string) *logrus.Logger {
	verbose = viper.GetString("verbose")

	// Create a new logrus logger
	l := logrus.New()
	l.SetOutput(os.Stderr)
	lvl, err := logrus.ParseLevel(v)
	if err != nil {
		log.Fatalf("error parsing level %v: %v", v, err)
	}
	l.SetLevel(lvl)
	return l
}

func main() {
	l := getLogger("info")

	viper.SetEnvPrefix(appName) // Environment variables prefixed with GIVME_
	viper.AutomaticEnv()        // Automatically bind environment variables

	var rootCmd = &cobra.Command{Use: appName}

	// Define global flags
	rootCmd.PersistentFlags().StringVarP(
		&workDir, "workdir", "w", ".", "Working directory")
	rootCmd.PersistentFlags().StringVarP(
		&file, "file", "f", "", "The archive path")
	rootCmd.PersistentFlags().StringVarP(
		&rootfs, "rootfs", "r", "/", "RootFS directory")
	rootCmd.PersistentFlags().StringVarP(
		&exclude, "exclude", "e", "", "Excluded directories")
	rootCmd.PersistentFlags().StringVarP(
		&exclude, "dotenv", "", "", ".env file")
	rootCmd.PersistentFlags().StringVarP(
		&exclude, "verbose", "v", "info", "Verbose level of output")

	// Bind flags to environment variables
	viper.BindPFlags(rootCmd.PersistentFlags())

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {

		// Set variables from flags or environment
		workDir = viper.GetString("workdir")
		file = viper.GetString("file")
		rootfs = viper.GetString("rootfs")
		exclude = viper.GetString("exclude")

		// Default tar file if not provided.
		if file == "" {
			file = filepath.Join(workDir, "snapshot.tar")
		}
		// Default .env file if not provided.
		if dotenv == "" {
			dotenv = filepath.Join(workDir, ".env")
		}

		// Ensure the work directory exists.
		if err := os.MkdirAll(workDir, 0755); err != nil {
			l.Errorf("Error creating work directory: %v", err)
			os.Exit(1)
		}
	}

	// Add subcommands for snapshot, restore, and cleanup.
	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "snapshot",
			Short: "Create a snapshot archive",
			Run: func(cmd *cobra.Command, args []string) {
				snapshotCmd(l)
			},
		},
		&cobra.Command{
			Use:   "restore",
			Short: "Restore from a snapshot archive",
			Run: func(cmd *cobra.Command, args []string) {
				restoreCmd(l, file)
			},
		},
		&cobra.Command{
			Use:   "cleanup",
			Short: "Clean up directories",
			Run: func(cmd *cobra.Command, args []string) {
				cleanupCmd(l)
			},
		},
		&cobra.Command{
			Use:   "download",
			Short: "Download container image",
			Args:  cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
			Run: func(cmd *cobra.Command, args []string) {
				downloadCmd(l, args[0]) // Pass the argument to the action
			},
		},
		&cobra.Command{
			Use:   "load",
			Short: "Load container image",
			Args:  cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
			Run: func(cmd *cobra.Command, args []string) {
				snapshotCmd(l)
				downloadCmd(l, args[0])
				cleanupCmd(l)
				tar := filepath.Join(workDir, util.Slugify(args[0]), "image.tar")
				restoreCmd(l, tar)
				imageEnv, err := image.GetEnv(l, args[0])
				if err != nil {
					l.Errorln(err)
					os.Exit(1)
				} else {
					fmt.Printf("%s\n", strings.Join(imageEnv, "\n"))
				}
				l.Infof("Image %s has been loaded!\n", args[0])
			},
		},
	)

	// Execute the root command.
	if err := rootCmd.Execute(); err != nil {
		l.Errorln(err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/list"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const appName = "givme"

var (
	target, workDir, source, exclude string
)

// buildExclusions generates a list of directories that should be excluded
// from operations such as snapshot creation or restoration.
func buildExclusions() ([]string, error) {
	mounts, err := util.GetMounts() // Get system mount points.
	if err != nil {
		return nil, err
	}

	// Append system directories and user-defined paths to exclusions.
	exclusions := append(mounts, "/proc", "/sys", "/dev", "/run",
		"/busybox", workDir, target)

	// Add user-specified excluded directories from environment variable.
	if exclude != "" {
		exclusions = append(exclusions, strings.FieldsFunc(
			exclude, func(r rune) bool {
				return r == ':' || r == ','
			})...)
	}

	return exclusions, nil
}

// runWithPreCheck is a helper function that runs the provided action function
// and handles error checking, printing a success or failure message.
func runWithPreCheck(cmd *cobra.Command, action func() error) {
	if err := action(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Println(strings.Title(cmd.Use) + " completed successfully.")
	}
}

// snapshotAction creates a tar archive of the source directory, excluding
// the directories specified in buildExclusions.
func snapshotAction() error {
	allExcludes, err := buildExclusions()
	if err != nil {
		return err
	}

	var paths []string
	if err := list.ListPaths(source, allExcludes, &paths); err != nil {
		return err
	}

	// Check if the target file already exists.
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("file %s already exists", target)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking file %s: %v", target, err)
	}

	// Create the tar archive.
	return archiver.Tar(paths, target)
}

// restoreAction extracts the contents of the tar archive to the source
// directory, while skipping directories listed in buildExclusions.
func restoreAction() error {
	allExcludes, err := buildExclusions()
	if err != nil {
		return err
	}

	return archiver.Untar(target, source, allExcludes)
}

// cleanupAction removes files and directories in the source directory,
// excluding the paths specified in buildExclusions.
func cleanupAction() error {
	allExcludes, err := buildExclusions()
	if err != nil {
		return err
	}

	var paths []string
	if err := list.ListPaths(source, allExcludes, &paths); err != nil {
		return err
	}

	return util.Rmrf(paths)
}

func main() {
	viper.SetEnvPrefix(appName) // Environment variables prefixed with GIVME_
	viper.AutomaticEnv()        // Automatically bind environment variables

	var rootCmd = &cobra.Command{Use: appName}

	// Define global flags
	rootCmd.PersistentFlags().StringVarP(
		&workDir, "workdir", "w", "", "Working directory")
	rootCmd.PersistentFlags().StringVarP(
		&target, "target", "t", "", "Target archive path")
	rootCmd.PersistentFlags().StringVarP(
		&source, "source", "s", "/", "Source directory")
	rootCmd.PersistentFlags().StringVarP(
		&exclude, "exclude", "e", "", "Excluded directories")

	// Bind flags to environment variables
	viper.BindPFlags(rootCmd.PersistentFlags())

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// Set variables from flags or environment
		workDir = viper.GetString("workdir")
		target = viper.GetString("target")
		source = viper.GetString("source")
		exclude = viper.GetString("exclude")

		// Default target file if not provided.
		if target == "" {
			target = filepath.Join(workDir, "image.tar")
		}

		// If workDir is not set, use the directory of the executable.
		if workDir == "" {
			workDir = util.GetExecDir()
		}

		// Ensure the work directory exists.
		if err := os.MkdirAll(workDir, 0755); err != nil {
			fmt.Printf("Error creating work directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Add subcommands for snapshot, restore, and cleanup.
	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "snapshot",
			Short: "Create a snapshot archive",
			Run: func(cmd *cobra.Command, args []string) {
				runWithPreCheck(cmd, snapshotAction)
			},
		},
		&cobra.Command{
			Use:   "restore",
			Short: "Restore from a snapshot archive",
			Run: func(cmd *cobra.Command, args []string) {
				runWithPreCheck(cmd, restoreAction)
			},
		},
		&cobra.Command{
			Use:   "cleanup",
			Short: "Clean up directories",
			Run: func(cmd *cobra.Command, args []string) {
				runWithPreCheck(cmd, cleanupAction)
			},
		},
	)

	// Execute the root command.
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

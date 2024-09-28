package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const appName = "givme"

var (
	dotenv, file, workDir, rootfs, exclude string
)

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

func main() {
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
				runWithPreCheck(cmd, snapshotCmd)
			},
		},
		&cobra.Command{
			Use:   "restore",
			Short: "Restore from a snapshot archive",
			Run: func(cmd *cobra.Command, args []string) {
				restoreCmd(file)
				fmt.Println("System successfully restored.")
			},
		},
		&cobra.Command{
			Use:   "cleanup",
			Short: "Clean up directories",
			Run: func(cmd *cobra.Command, args []string) {
				runWithPreCheck(cmd, cleanupCmd)
			},
		},
		&cobra.Command{
			Use:   "download",
			Short: "Download container image",
			Args:  cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
			Run: func(cmd *cobra.Command, args []string) {
				name := args[0] // Access the argument
				runWithPreCheck(cmd, func() error {
					return downloadCmd(name) // Pass the argument to the action
				})
			},
		},
		&cobra.Command{
			Use:   "unpack",
			Short: "Unpack container image",
			Args:  cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
			Run: func(cmd *cobra.Command, args []string) {
				snapshotCmd()
				downloadCmd(args[0])
				cleanupCmd()
				tar := filepath.Join(workDir, util.Slugify(args[0]), "image.tar")
				restoreCmd(tar)
				fmt.Fprintf(os.Stderr, "Image %s successfully unpacked!\n", args[0])
				imageEnv, _ := image.GetEnv(args[0])
				fmt.Printf("%s\n", strings.Join(imageEnv, "\n"))
			},
		},
	)

	// Execute the root command.
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

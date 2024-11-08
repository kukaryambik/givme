package cmd

import (
	"fmt"
	"syscall"

	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func ExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "exec [flags] IMAGE [cmd]...",
		Aliases: []string{"e"},
		Short:   "Exec a command in the container",
		Args:    cobra.MinimumNArgs(1), // Ensure exactly 1 argument is provided
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Image = args[0]
			opts.Cmd = args[1:]
			cmd.SilenceUsage = true
			return opts.Exec()
		},
	}

	cmd.Flags().BoolVar(
		&opts.Update, "update", opts.Update, "Update the image instead of using existing file")
	cmd.Flags().BoolVar(
		&opts.OverwriteEnv, "overwrite-env", opts.OverwriteEnv, "Overwrite current environment variables with new ones from the image")
	cmd.Flags().BoolVar(
		&opts.NoPurge, "no-purge", opts.NoPurge, "Do not purge the root directory before unpacking the image")
	cmd.Flags().StringArrayVar(
		&opts.Entrypoint, "entrypoint", opts.Entrypoint, "Entrypoint for the container")
	cmd.Flags().StringVarP(
		&opts.Cwd, "cwd", "w", opts.Cwd, "Working directory for the container")

	return cmd
}

func (opts *CommandOptions) Exec() error {

	img, err := opts.Extract()
	if err != nil {
		return err
	}

	// Get the image config
	logrus.Debugf("Fetching config file for image %s", img.Name)
	cfg, err := img.Config()
	if err != nil {
		return fmt.Errorf("error getting config from image %s: %v", img, err)
	}

	// Prepare the command
	command := opts.PrepareEntrypoint(&cfg.Config)

	// Prepare environment variables
	logrus.Info("Preparing environment variables")
	env, err := opts.PrepareEnvForExec(&cfg.Config)
	if err != nil {
		return err
	}

	// Change the working directory
	if err := syscall.Chdir(util.Coalesce(opts.Cwd, cfg.Config.WorkingDir, "/")); err != nil {
		return fmt.Errorf("invalid working directory %q: %v", opts.Cwd, err)
	}

	// Set the entrypoint
	entrypoint, err := envars.CoalesceWhich(env, command[0])
	if err != nil {
		return err
	}

	// Run the command
	return syscall.Exec(entrypoint, command, env)
}

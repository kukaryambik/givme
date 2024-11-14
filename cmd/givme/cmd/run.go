package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/kukaryambik/givme/pkg/proot"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func RunCmd() *cobra.Command {
	cmd := &cobra.Command{
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

	cmd.Flags().BoolVar(
		&opts.Update, "update", opts.Update, "Update the image instead of using existing file")
	cmd.Flags().BoolVar(
		&opts.OverwriteEnv, "overwrite-env", opts.OverwriteEnv, "Overwrite current environment variables with new ones from the image")
	cmd.Flags().StringArrayVar(
		&opts.Entrypoint, "entrypoint", opts.Entrypoint, "Entrypoint for the container")
	cmd.Flags().StringVarP(
		&opts.Cwd, "cwd", "w", opts.Cwd, "Working directory for the container")
	cmd.Flags().StringVarP(&opts.RunChangeID, "change-id", "u", opts.RunChangeID, "UID:GID for the container")
	cmd.Flags().StringArrayVarP(
		&opts.RunProotBinds, "proot-bind", "b", opts.RunProotBinds, "Mount host path to the container")
	cmd.Flags().BoolVar(
		&opts.RunRemoveAfter, "rm", opts.RunRemoveAfter, "Remove the rootfs directory after running the command")
	cmd.Flags().StringVar(
		&opts.RunName, "name", opts.RunName, "The name of the container")
	cmd.Flags().StringVar(
		&opts.RunProotFlags, "proot-flags", opts.RunProotFlags, "Additional flags for proot")
	cmd.Flags().MarkHidden("proot-flags")
	cmd.Flags().StringVar(
		&opts.RunProotBin, "proot-bin", opts.RunProotBin, "Path to the proot binary")

	return cmd
}

func (opts *CommandOptions) Run() error {

	// Get an image
	img, err := opts.Save()
	if err != nil {
		return err
	}

	// Get the image config
	imgConf, err := img.Config()
	if err != nil {
		return err
	}
	cfg := imgConf.Config

	// Prepare the command
	command := opts.PrepareEntrypoint(&cfg)

	// Prepare environment variables
	logrus.Info("Preparing environment variables")
	env, err := opts.PrepareEnvForExec(&cfg)
	if err != nil {
		return err
	}

	// Set the rootfs directory
	if opts.RunName == "" {
		bytes := make([]byte, 6)
		if _, err := rand.Read(bytes); err != nil {
			return err
		}
		opts.RunName = hex.EncodeToString(bytes)
	}
	name := util.Slugify(opts.RunName)
	opts.RootFS = filepath.Join(opts.Workdir, "rootfs", name)
	logrus.Infof("Using %q as rootfs", opts.RootFS)

	// Remove the rootfs
	if opts.RunRemoveAfter {
		defer func() error {
			logrus.Infof("Removing rootfs '%s'", opts.RootFS)
			return os.RemoveAll(opts.RootFS)
		}()
	}

	// Untar the filesystem
	entries, err := os.ReadDir(opts.RootFS)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(entries) == 0 {
		if err := image.Extract(img, opts.RootFS); err != nil {
			return err
		}
	}

	// Create the proot command
	prootConf := proot.ProotConf{
		BinPath:    util.Coalesce(opts.RunProotBin, filepath.Join(util.GetExecDir(), "proot")),
		Binds:      opts.RunProotBinds,
		Command:    command,
		RootFS:     opts.RootFS,
		ChangeID:   util.Coalesce(opts.RunChangeID, cfg.User, "0:0"),
		Workdir:    util.Coalesce(opts.Cwd, cfg.WorkingDir, "/"),
		Env:        env,
		ExtraFlags: strings.Split(strings.TrimSpace(opts.RunProotFlags), " "),
		MixedMode:  true,
		TmpDir:     defaultCacheDir(),
		KillOnExit: true,
	}

	// Add mounts
	ignores := paths.Ignore(opts.IgnorePaths).AddPaths(opts.Workdir)
	logrus.Info("Ignored exclusions: ", ignores.Exclusions)
	for _, e := range ignores.Exclusions {
		realPath := filepath.Join(opts.RootFS, e)
		if err := os.MkdirAll(realPath, os.ModePerm); err != nil {
			return err
		}
		prootConf.Binds = append(prootConf.Binds, fmt.Sprintf("%s:%s", realPath, e))
	}
	ignores.Exclusions = nil
	binds, err := ignores.List()
	if err != nil {
		return err
	}
	prootConf.Binds = append(prootConf.Binds, binds...)

	// Create the proot command and run it
	cmd := prootConf.Cmd()
	logrus.Debug(cmd.Args)

	logrus.Info("Running proot")

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running proot: %v", err)
	}

	return nil
}

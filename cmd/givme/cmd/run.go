package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/kukaryambik/givme/pkg/proot"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func (opts *CommandOptions) run() error {

	if opts.RunName == "" {
		bytes := make([]byte, 6)
		if _, err := rand.Read(bytes); err != nil {
			return err
		}
		opts.RunName = hex.EncodeToString(bytes)
	}
	name := util.Slugify(opts.RunName)

	// Get the image
	img, err := opts.save()
	if err != nil {
		return err
	}

	// Set the rootfs directory
	if strings.Trim(opts.RootFS, "/") == "" {
		opts.RootFS = filepath.Join(opts.Workdir, "rootfs", name)
		logrus.Infof("Using '%s' as rootfs", opts.RootFS)
	}

	// Configure ignored paths
	ignoreConf := paths.Ignore(opts.IgnorePaths).ExclFromList(opts.RootFS)
	ignores, err := ignoreConf.AddPaths(opts.Workdir).List()
	if err != nil {
		return err
	}

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
		logrus.Infof("Extracting filesystem to '%s'", opts.RootFS)
		reader, writer := io.Pipe()
		go func() {
			if err := crane.Export(img.Image, writer); err != nil {
				writer.CloseWithError(err)
				return
			}
			writer.Close()
		}()

		if err := archiver.Untar(reader, opts.RootFS, ignores); err != nil {
			return err
		}
	}

	// Get the image config
	imgConf, err := img.Config()
	if err != nil {
		return err
	}
	cfg := imgConf.Config

	// Prepare environment variables
	current := envars.ToMap(os.Environ())
	new := envars.ToMap(cfg.Env)
	old, err := envars.FromFile(new, defaultDotEnvFile(), false)
	if err != nil {
		return err
	}

	diff := envars.Uniq(false, current, old)
	env := envars.Merge(new, diff)
	if opts.OverwriteEnv {
		env = envars.Merge(diff, new)
	}
	env["PATH"] = strings.Trim(new["PATH"]+":"+util.GetExecDir(), ": ")

	// Create the proot command
	prootConf := proot.ProotConf{
		BinPath:    filepath.Join(util.GetExecDir(), "proot"),
		RootFS:     opts.RootFS,
		ChangeID:   util.Coalesce(opts.RunChangeID, cfg.User, "0:0"),
		Workdir:    util.Coalesce(opts.RunCwd, cfg.WorkingDir, "/"),
		Env:        envars.ToSlice(false, env),
		ExtraFlags: strings.Split(strings.TrimSpace(opts.RunProotFlags), " "),
		MixedMode:  true,
		TmpDir:     opts.Workdir,
		KillOnExit: true,
	}

	// add volumes & mounts
	prootConf.Mounts = slices.Concat(opts.RunMounts, ignores)
	for v := range cfg.Volumes {
		oldpath := filepath.Join(opts.RootFS, v)
		newpath := filepath.Join(defaultCacheDir(), fmt.Sprintf("vol_%s_%s", name, util.Slugify(v)))
		if len(entries) == 0 {
			if err := os.Rename(oldpath, newpath); err != nil {
				return fmt.Errorf("error renaming %s to %s: %v", oldpath, newpath, err)
			}
		}
		f := newpath + ":" + v
		prootConf.Mounts = append(prootConf.Mounts, f)
	}

	// add command
	shell := util.Coalesce(util.CleanList(cfg.Shell), []string{"/bin/sh", "-c"})
	var args []string
	if len(opts.RunEntrypoint) > 0 {
		args = append(opts.RunEntrypoint[len(opts.RunEntrypoint)-1:], opts.Cmd...)
	} else {
		args = append(
			util.CleanList(cfg.Entrypoint),
			util.Coalesce(util.CleanList(opts.Cmd), util.CleanList(cfg.Cmd))...,
		)
	}
	args = append([]string{"exec"}, util.Coalesce(util.CleanList(args), shell[:1])...)
	prootConf.Command = util.CleanList(append(shell, strings.Join(args, " ")))

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

package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/kukaryambik/givme/pkg/proot"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

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
		if err := image.Extract(img, opts.RootFS, ignores...); err != nil {
			return err
		}
	}

	// Create the proot command
	prootConf := proot.ProotConf{
		BinPath:    util.Coalesce(opts.RunProotBin, filepath.Join(util.GetExecDir(), "proot")),
		Command:    command,
		RootFS:     opts.RootFS,
		ChangeID:   util.Coalesce(opts.RunChangeID, cfg.User, "0:0"),
		Workdir:    util.Coalesce(opts.Cwd, cfg.WorkingDir, "/"),
		Env:        env,
		ExtraFlags: strings.Split(strings.TrimSpace(opts.RunProotFlags), " "),
		MixedMode:  true,
		TmpDir:     opts.Workdir,
		KillOnExit: true,
	}

	// add volumes & mounts
	prootConf.Binds = slices.Concat(opts.RunProotBinds, ignores)
	for v := range cfg.Volumes {
		oldpath := filepath.Join(opts.RootFS, v)
		newpath := filepath.Join(defaultCacheDir(), fmt.Sprintf("vol_%s_%s", name, util.Slugify(v)))
		if len(entries) == 0 {
			if err := os.Rename(oldpath, newpath); err != nil {
				return fmt.Errorf("error renaming %s to %s: %v", oldpath, newpath, err)
			}
		}
		f := newpath + ":" + v
		prootConf.Binds = append(prootConf.Binds, f)
	}

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

package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/kukaryambik/givme/pkg/proot"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func (opts *CommandOptions) run() error {

	// Get the image
	img, err := opts.save()
	if err != nil {
		return err
	}

	// Get image slug
	imageSlug, err := image.GetNameSlug(opts.Image)
	if err != nil {
		return err
	}

	// Set the rootfs directory
	if strings.Trim(opts.RootFS, "/") == "" {
		opts.RootFS = filepath.Join(opts.Workdir, "rootfs_"+imageSlug)
		logrus.Infof("Using '%s' as rootfs", opts.RootFS)
	}

	// Configure ignored paths
	ignoreConf := paths.Ignore(opts.IgnorePaths).ExclFromList(opts.RootFS)
	ignores, err := ignoreConf.AddPaths(opts.Workdir).List()
	if err != nil {
		return err
	}

	// Purge the rootfs
	if !opts.NoPurge {
		logrus.Infof("Purging rootfs '%s'", opts.RootFS)
		if err := paths.Rmrf(opts.RootFS, ignores); err != nil {
			return err
		}
	}

	logrus.Infof("Extracting filesystem to '%s'", opts.RootFS)

	// Untar the filesystem
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

	// Get the image config
	imgConf, err := img.Config()
	if err != nil {
		return err
	}
	cfg := imgConf.Config

	env := envars.AddToPath(cfg.Env, util.GetExecDir())

	// Create the proot command
	prootConf := proot.ProotConf{
		BinPath:    filepath.Join(util.GetExecDir(), "proot"),
		RootFS:     opts.RootFS,
		ChangeID:   util.Coalesce(opts.ProotUser, cfg.User, "0:0"),
		Workdir:    util.Coalesce(opts.ProotCwd, cfg.WorkingDir, "/"),
		Env:        slices.Concat(os.Environ(), env),
		ExtraFlags: opts.ProotFlags,
		MixedMode:  true,
		TmpDir:     opts.Workdir,
		KillOnExit: true,
	}

	// add volumes & mounts
	prootConf.Mounts = slices.Concat(opts.ProotMounts, ignores)
	for v := range cfg.Volumes {
		oldpath := filepath.Join(opts.RootFS, v)
		newpath := filepath.Join(defaultCacheDir(), "vol_"+imageSlug+util.Slugify(v))
		if err := os.Rename(oldpath, newpath); err != nil {
			return fmt.Errorf("error renaming %s to %s: %v", oldpath, newpath, err)
		}
		f := newpath + ":" + v
		prootConf.Mounts = append(prootConf.Mounts, f)
	}

	// add command
	prootConf.Command = slices.Concat(
		cfg.Shell,
		util.Coalesce([]string{opts.ProotEntrypoint}, cfg.Entrypoint),
		util.Coalesce(opts.Cmd, cfg.Cmd),
	)

	// Create the proot command and run it
	cmd := prootConf.Cmd()
	logrus.Debug(cmd.Args)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	logrus.Info("Running proot")

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running proot: %v", err)
	}

	return nil
}

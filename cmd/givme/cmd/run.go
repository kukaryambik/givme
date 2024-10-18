package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/kukaryambik/givme/pkg/proot"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func run(opts *CommandOptions) error {

	// Get the image
	img, err := save(opts)
	if err != nil {
		return err
	}

	// Create the image workspace
	dir, err := image.MkImageDir(opts.Workdir, opts.Image)
	if err != nil {
		return err
	}

	// Set the rootfs directory
	if strings.Trim(opts.RootFS, "/") == "" {
		opts.RootFS = filepath.Join(dir, "rootfs")
	}

	// Configure ignored paths
	ignoreConf := paths.Ignore(opts.IgnorePaths).ExclFromList(opts.RootFS)
	ignores, err := ignoreConf.AddPaths(opts.Workdir).List()
	if err != nil {
		return err
	}

	// Get the image config
	imgConf, err := img.Config()
	if err != nil {
		return err
	}
	cfg := imgConf.Config

	// Create the proot command
	prootConf := proot.ProotConf{
		BinPath:    filepath.Join(paths.GetExecDir(), "proot"),
		RootFS:     opts.RootFS,
		ChangeID:   util.Coalesce(opts.ProotUser, cfg.User, "0:0"),
		Workdir:    util.Coalesce(opts.ProotCwd, cfg.WorkingDir, "/"),
		Env:        cfg.Env,
		ExtraFlags: opts.ProotFlags,
		MixedMode:  true,
		TmpDir:     opts.Workdir,
		KillOnExit: true,
	}

	// add volumes & mounts
	prootConf.Mounts = slices.Concat(opts.ProotMounts, ignores)
	for v := range cfg.Volumes {
		tmpDir := filepath.Join(dir, "vol_"+util.Slugify(v))
		if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
			return fmt.Errorf("error creating directory %s: %v", dir, err)
		}
		f := tmpDir + ":" + v
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
	logrus.Debugln(cmd.Args)

	// Export the image filesystem to the tar file
	tmpFS := filepath.Join(dir, "fs.tar")
	defer os.Remove(tmpFS)
	if err := img.Export(tmpFS); err != nil {
		return err
	}

	// Clean up the rootfs
	if opts.Cleanup {
		if err := cleanup(opts); err != nil {
			return err
		}
	}

	// Untar the filesystem
	if err := archiver.Untar(tmpFS, opts.RootFS, ignores); err != nil {
		return err
	}

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running proot: %v", err)
	}

	return nil
}

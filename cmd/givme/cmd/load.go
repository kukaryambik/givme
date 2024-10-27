package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func load(opts *CommandOptions) (*image.Image, error) {

	img, err := save(opts)
	if err != nil {
		return nil, err
	}

	dir, err := image.MkImageDir(opts.Workdir, opts.Image)
	if err != nil {
		return nil, err
	}
	fs := filepath.Join(dir, "fs.tar")
	defer os.Remove(fs)

	if err := img.Export(fs); err != nil {
		return nil, err
	}

	// Clean up the rootfs
	if opts.Cleanup {
		if err := cleanup(opts); err != nil {
			return nil, err
		}
	}

	// Configure ignored paths
	ignoreConf := paths.Ignore(opts.IgnorePaths).ExclFromList(opts.RootFS)
	ignores, err := ignoreConf.AddPaths(opts.Workdir).List()
	if err != nil {
		return nil, err
	}

	if err := archiver.Untar(fs, opts.RootFS, ignores); err != nil {
		return nil, err
	}

	logrus.Debugf("Fetching config file for image %s", img.Name)
	cfg, err := img.Config()
	if err != nil {
		return nil, fmt.Errorf("error getting config from image %s: %v", img, err)
	}

	logrus.Infof("Image %s has been loaded!\n", opts.Image)

	envs := util.PrepareEnv(cfg.Config.Env)

	fmt.Printf(
		"# Environments variables for %s:\n%s\n",
		opts.Image,
		strings.Join(envs, "\n"),
	)

	return img, nil
}

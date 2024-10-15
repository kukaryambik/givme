package cmd

import (
	"path/filepath"

	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
)

func export(opts *CommandOptions) error {

	dir, err := image.MkImageDir(opts.Workdir, opts.Image)
	if err != nil {
		return err
	}

	if opts.TarFile == "" {
		opts.TarFile = filepath.Join(dir, "fs.tar")
	}

	if paths.IsFileExists(opts.TarFile) {
		logrus.Warnf("File %s already exists. Skipping export.", opts.TarFile)
		return nil
	}

	img, err := image.Load(opts.Image)
	if err != nil {
		return err
	}

	if err := img.Export(opts.TarFile); err != nil {
		return err
	}

	logrus.Infof("Image %s has been exported to %s!\n", opts.Image, opts.TarFile)
	return nil
}

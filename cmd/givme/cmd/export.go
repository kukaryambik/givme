package cmd

import (
	"path/filepath"

	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func export(opts *CommandOptions) error {

	saveOpts := *opts
	saveOpts.TarFile = ""
	img, err := save(&saveOpts)
	if err != nil {
		return err
	}

	if opts.TarFile == "" {
		imgSlug := util.Slugify(img.Name)
		opts.TarFile = filepath.Join(opts.Workdir, imgSlug+".fs.tar")
	}

	if util.IsFileExists(opts.TarFile) {
		logrus.Warnf("File %s already exists. Skipping export.", opts.TarFile)
		return nil
	}

	if err := img.Export(opts.TarFile); err != nil {
		return err
	}

	logrus.Infof("Image %s has been exported to %s!\n", opts.Image, opts.TarFile)
	return nil
}

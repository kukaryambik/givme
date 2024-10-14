package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func load(opts *CommandOptions) (*image.Image, error) {

	img, err := save(opts)
	if err != nil {
		return nil, err
	}

	tmpFS := filepath.Join(opts.Workdir, ".fs_"+util.Slugify(opts.Image)+".tar")
	defer os.Remove(tmpFS)

	if err := img.Export(tmpFS); err != nil {
		return nil, err
	}

	logrus.Debugf("Fetching config file for image %s", img.Name)
	cfg, err := img.Config()
	if err != nil {
		return nil, fmt.Errorf("error getting config from image %s: %v", img, err)
	}

	if err := cleanup(opts); err != nil {
		return nil, err
	}

	if err := archiver.Untar(tmpFS, opts.RootFS, opts.Exclusions); err != nil {
		return nil, err
	}

	env := cfg.Config.Env
	if opts.Eval {
		fmt.Println(strings.Join(env, "\n"))
	}

	logrus.Infof("Image %s has been loaded!\n", opts.Image)
	return img, nil
}

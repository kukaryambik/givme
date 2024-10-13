package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func getenv(opts *CommandOptions) error {

	img, err := save(opts)
	if err != nil {
		return err
	}

	if opts.DotenvFile == "" {
		imgSlug := util.Slugify(img.Name)
		opts.DotenvFile = filepath.Join(opts.Workdir, imgSlug+".env")
	}

	logrus.Debugf("Fetching config file for image %s", img)
	cfg, err := img.Config()
	if err != nil {
		return fmt.Errorf("error getting config from image %s: %v", img, err)
	}

	logrus.Infoln(img.Name)

	env := cfg.Config.Env

	if err := envars.SaveToFile(env, opts.DotenvFile); err != nil {
		return err
	}

	if opts.Eval {
		fmt.Println(strings.Join(env, "\n"))
	}

	logrus.Infof("Environment variables for %s has been saved to %s!\n", opts.Image, opts.DotenvFile)
	return nil
}

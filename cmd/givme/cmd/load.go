package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func load(conf *CommandOptions) error {

	img, err := save(conf)
	if err != nil {
		return err
	}

	tmpFS := filepath.Join(conf.Workdir, ".fs_"+util.Slugify(conf.Image)+".tar")
	defer os.Remove(tmpFS)

	if err := img.Export(tmpFS); err != nil {
		return err
	}

	if err := cleanup(conf); err != nil {
		return err
	}

	if err := archiver.Untar(tmpFS, conf.RootFS, conf.Exclusions); err != nil {
		return err
	}

	logrus.Debugf("Fetching config file for image %s", img)
	cfg, err := img.Config()
	if err != nil {
		return fmt.Errorf("error getting config from image %s: %v", img, err)
	}
	env := cfg.Config.Env
	if conf.Eval {
		fmt.Println(strings.Join(env, "\n"))
	}

	logrus.Infof("Image %s has been loaded!\n", conf.Image)
	return nil
}

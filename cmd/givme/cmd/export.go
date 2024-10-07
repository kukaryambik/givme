package cmd

import (
	"fmt"

	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/sirupsen/logrus"
)

func export(conf *CommandOptions) error {

	img, err := save(conf)
	if err != nil {
		return err
	}

	if err := img.Export(conf.TarFile); err != nil {
		return err
	}

	logrus.Debugf("Fetching config file for image %s", img)
	cfg, err := img.Config()
	if err != nil {
		return fmt.Errorf("error getting config from image %s: %v", img, err)
	}

	if conf.DotenvFile != "" {
		envars.SaveToFile(cfg.Config.Env, conf.DotenvFile)
	}

	logrus.Infof("Image %s has been loaded!\n", conf.Image)
	return nil
}

package cmd

import (
	"github.com/sirupsen/logrus"
)

func load(conf *CommandOptions) error {

	_, err := export(conf)
	if err != nil {
		return err
	}

	if err := restore(conf); err != nil {
		return err
	}

	logrus.Infof("Image %s has been loaded!\n", conf.Image)
	return nil
}

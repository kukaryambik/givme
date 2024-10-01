package cmd

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/sirupsen/logrus"
)

func export(conf *CommandOptions) ([]string, error) {
	logrus.Debugf("Starting download of image: %s", conf.Image)
	logrus.Debug(conf)

	auth := &authn.Basic{
		Username: conf.RegistryUsername,
		Password: conf.RegistryPassword,
	}

	// Pull the image
	img, err := image.Pull(auth, conf.Image)
	if err != nil {
		return nil, err
	}

	logrus.Debugf("Fetching config file for image %s", img)
	cfg, err := img.GetConfig(conf.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("error getting config from image %s: %v", img, err)
	}
	logrus.Infof("%s config has been saved to %s", conf.Image, conf.ConfigFile)

	env := cfg.Config.Env

	if err := envars.SaveToFile(env, conf.DotenvFile); err != nil {
		return env, err
	}
	logrus.Infof("%s dotenv has been saved to %s", conf.Image, conf.DotenvFile)

	logrus.Debugf("Getting filesystem of image %s and saving to %s", conf.Image, conf.TarFile)
	if err := img.GetFS(conf.TarFile); err != nil {
		return env, fmt.Errorf("error getting image %s: %v", conf.Image, err)
	}
	logrus.Infof("%s filesystem has been saved to %s", conf.Image, conf.TarFile)

	return env, nil
}

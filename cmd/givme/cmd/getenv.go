package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/sirupsen/logrus"
)

func (opts *CommandOptions) getenv() error {
	logrus.Infof("Loading image for %s", opts.Image)

	imageSlug, err := image.GetNameSlug(opts.Image)
	if err != nil {
		return err
	}
	if opts.TarFile == "" {
		opts.TarFile = filepath.Join(defaultImagesDir(), imageSlug+".tar")
	}

	conf := &image.GetConf{
		File:             opts.TarFile,
		Image:            opts.Image,
		RegistryMirror:   opts.RegistryMirror,
		RegistryPassword: opts.RegistryPassword,
		RegistryUsername: opts.RegistryUsername,
		CacheDir:         defaultLayersDir(),
		Update:           opts.Update,
		Save:             false,
	}

	img, err := conf.Get()
	if err != nil {
		return err
	}

	logrus.Info("Fetching config")
	cfg, err := img.Config()
	if err != nil {
		return fmt.Errorf("error getting config from image %s: %v", img, err)
	}

	// Prepare environment variables
	env, err := godotenv.Marshal(envars.Split(cfg.Config.Env))
	if err != nil {
		return fmt.Errorf("error marshalling environment variables: %v", err)
	}

	fmt.Println(env)

	return err
}

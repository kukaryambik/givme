package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

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

	var env string
	if opts.IntactEnv {
		env, _ = godotenv.Marshal(envars.Split(cfg.Config.Env))
	} else {
		list := envars.PrepareEnv(defaultDotEnvFile(), opts.RedirectOutput, cfg.Config.Env)
		env = strings.Join(list, "\n")
	}

	fmt.Println(env)

	return err
}

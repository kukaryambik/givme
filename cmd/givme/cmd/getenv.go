package cmd

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
)

func (opts *CommandOptions) getenv() error {

	var (
		img *image.Image
		err error
	)
	if paths.FileExists(opts.Image) {
		img, err = image.Load(opts.Image)
		if err != nil {
			return err
		}
	} else {
		imgConf := &image.GetConf{
			Image:            opts.Image,
			RegistryMirror:   opts.RegistryMirror,
			RegistryPassword: opts.RegistryPassword,
			RegistryUsername: opts.RegistryUsername,
			CacheDir:         defaultLayersDir(),
		}
		img, err = imgConf.Pull()
		if err != nil {
			return err
		}
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

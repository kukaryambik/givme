package cmd

import (
	"path/filepath"

	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func load(conf *CommandOptions) error {
	if err := snapshot(conf); err != nil {
		return err
	}

	imgSlug := util.Slugify(conf.Image)
	imgConfigFile := filepath.Join(conf.Workdir, imgSlug+".json")
	imgDotenvFile := filepath.Join(conf.Workdir, imgSlug+".env")
	imgTarFile := filepath.Join(conf.Workdir, imgSlug+".tar")
	_, err := export(&CommandOptions{
		Image:      conf.Image,
		TarFile:    imgTarFile,
		DotenvFile: imgDotenvFile,
		ConfigFile: imgConfigFile,
	})
	if err != nil {
		return err
	}

	if err := restore(&CommandOptions{
		TarFile:    imgTarFile,
		DotenvFile: imgDotenvFile,
		RootFS:     conf.RootFS,
		Exclusions: conf.Exclusions,
	}); err != nil {
		return err
	}

	logrus.Infof("Image %s has been loaded!\n", conf.Image)
	return nil
}

package cmd

import (
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func save(conf *CommandOptions) (*image.Image, error) {
	auth := &authn.Basic{
		Username: conf.RegistryUsername,
		Password: conf.RegistryPassword,
	}

	if util.IsFileExists(conf.Image) {
		conf.TarFile = conf.Image
	} else if conf.TarFile != "" && !util.IsFileExists(conf.TarFile) {
		// Pull the image
		img, err := image.Pull(auth, conf.Image, conf.RegistryMirror)
		if err != nil {
			return nil, err
		}

		err = util.Retry(conf.Retry, 5*time.Second, func() error {
			return img.Save(conf.TarFile)
		})
		if err != nil {
			return nil, err
		}
		logrus.Infof("Image %s has been saved to %s", conf.Image, conf.TarFile)
	}

	// Load the image
	img, err := image.Load(conf.TarFile)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Using file %s", conf.TarFile)
	return img, nil
}

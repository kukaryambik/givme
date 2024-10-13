package cmd

import (
	"path/filepath"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func save(opts *CommandOptions) (*image.Image, error) {
	auth := &authn.Basic{
		Username: opts.RegistryUsername,
		Password: opts.RegistryPassword,
	}

	imgSlug := util.Slugify(opts.Image)
	if opts.TarFile == "" {
		opts.TarFile = filepath.Join(opts.Workdir, imgSlug+".tar")
	}

	if util.IsFileExists(opts.Image) {
		opts.TarFile = opts.Image
	} else if opts.TarFile != "" && !util.IsFileExists(opts.TarFile) {
		// Pull the image
		img, err := image.Pull(auth, opts.Image, opts.RegistryMirror)
		if err != nil {
			return nil, err
		}

		err = util.Retry(opts.Retry, 5*time.Second, func() error {
			return img.Save(opts.TarFile)
		})
		if err != nil {
			return nil, err
		}
		logrus.Infof("Image %s has been saved to %s", opts.Image, opts.TarFile)
	}

	// Load the image
	img, err := image.Load(opts.TarFile)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Using file %s", opts.TarFile)
	return img, nil
}

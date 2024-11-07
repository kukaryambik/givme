package cmd

import (
	"path/filepath"

	"github.com/kukaryambik/givme/pkg/image"
	"github.com/sirupsen/logrus"
)

func (opts *CommandOptions) Save() (*image.Image, error) {

	logrus.Infof("Loading image for %s", opts.Image)

	imageSlug, err := image.GetNameSlug(opts.Image)
	if err != nil {
		return nil, err
	}
	if opts.TarFile == "" {
		opts.TarFile = filepath.Join(defaultImagesDir(), imageSlug+".tar")
	}

	img := &image.GetConf{
		File:             opts.TarFile,
		Image:            opts.Image,
		RegistryMirror:   opts.RegistryMirror,
		RegistryPassword: opts.RegistryPassword,
		RegistryUsername: opts.RegistryUsername,
		CacheDir:         defaultLayersDir(),
		Update:           opts.Update,
		Save:             true,
	}

	return img.Get()
}

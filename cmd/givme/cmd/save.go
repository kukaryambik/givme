package cmd

import (
	"path/filepath"

	"github.com/kukaryambik/givme/pkg/image"
)

func save(opts *CommandOptions) (*image.Image, error) {

	dir, err := image.MkImageDir(opts.Workdir, opts.Image)
	if err != nil {
		return nil, err
	}

	if opts.TarFile == "" {
		opts.TarFile = filepath.Join(dir, "image.tar")
	}

	img := &image.GetConf{
		File:             opts.TarFile,
		Image:            opts.Image,
		RegistryMirror:   opts.RegistryMirror,
		RegistryPassword: opts.RegistryPassword,
		RegistryUsername: opts.RegistryUsername,
		Retry:            opts.Retry,
		CacheDir:         filepath.Join(opts.Workdir, "cache"),
	}

	return img.Get()
}

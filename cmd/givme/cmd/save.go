package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/kukaryambik/givme/pkg/image"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func SaveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "save [flags] IMAGE",
		Aliases: []string{"download", "pull"},
		Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
		Short:   "Save image to tar archive",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Image = args[0]
			opts.Update = true
			cmd.SilenceUsage = true
			img, err := opts.Save()
			fmt.Println(img.File)
			return err
		},
	}

	cmd.Flags().StringVarP(&opts.TarFile, "tar-file", "f", "", "Path to the tar file")
	cmd.MarkFlagFilename("tar-file", ".tar")

	return cmd
}

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

package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func getenvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "getenv [flags] IMAGE",
		Aliases: []string{"env"},
		Short:   "Get environment variables from image",
		Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Image = args[0]
			cmd.SilenceUsage = true
			return opts.Getenv()
		},
	}

	return cmd
}

func (opts *CommandOptions) Getenv() error {
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
	env, err := godotenv.Marshal(envars.ToMap(cfg.Config.Env))
	if err != nil {
		return fmt.Errorf("error marshalling environment variables: %v", err)
	}

	fmt.Println(env)

	return err
}

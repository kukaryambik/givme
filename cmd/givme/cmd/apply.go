package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/joho/godotenv"
	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
)

func (opts *CommandOptions) apply() (*image.Image, error) {

	img, err := opts.save()
	if err != nil {
		return nil, err
	}

	// Configure ignored paths
	ignoreConf := paths.Ignore(opts.IgnorePaths).ExclFromList(opts.RootFS)
	ignores, err := ignoreConf.AddPaths(opts.Workdir).List()
	if err != nil {
		return nil, err
	}

	// Clean up the rootfs
	if !opts.NoPurge {
		logrus.Infof("Purging rootfs '%s'", opts.RootFS)
		if err := paths.Rmrf(opts.RootFS, ignores); err != nil {
			return nil, err
		}
	}

	logrus.Infof("Extracting filesystem to '%s'", opts.RootFS)

	// Untar the filesystem
	reader, writer := io.Pipe()
	go func() {
		if err := crane.Export(img.Image, writer); err != nil {
			writer.CloseWithError(err)
			return
		}
		writer.Close()
	}()

	if err := archiver.Untar(reader, opts.RootFS, ignores); err != nil {
		return nil, err
	}

	logrus.Debugf("Fetching config file for image %s", img.Name)
	cfg, err := img.Config()
	if err != nil {
		return nil, fmt.Errorf("error getting config from image %s: %v", img, err)
	}

	logrus.Info("Image applied")

	if opts.RedirectOutput {
		var env string
		if opts.IntactEnv {
			env, _ = godotenv.Marshal(envars.Split(cfg.Config.Env))
		} else {
			list := envars.PrepareEnv(defaultDotEnvFile(), true, cfg.Config.Env)
			env = strings.Join(list, "\n")
		}
		fmt.Println(env)
	}

	return img, nil
}

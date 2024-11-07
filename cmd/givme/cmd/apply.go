package cmd

import (
	"fmt"

	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func (opts *CommandOptions) Apply() (*image.Image, error) {

	img, err := opts.Extract()
	if err != nil {
		return nil, err
	}

	logrus.Debugf("Fetching config file for image %s", img.Name)
	cfg, err := img.Config()
	if err != nil {
		return nil, fmt.Errorf("error getting config from image %s: %v", img, err)
	}

	logrus.Info("Preparing environment variables")

	outRedirected := util.IsOutRedirected()
	if !outRedirected {
		logrus.Warnf(
			"Output is not redirected!\n"+
				"It is strongly recommended to use this command in conjunction with source or eval. For example:\n"+
				"– `source <(%s apply %s)`\n"+
				"– `eval $(%s apply %s)`",
			AppName, opts.Image, AppName, opts.Image,
		)
		logrus.Info("Image environment variables will not be saved to a file")
	}

	// Prepare environment variables
	env, err := opts.PrepareEnvForEval(&cfg.Config, outRedirected)
	if err != nil {
		return nil, err
	}

	fmt.Println(env)

	return img, nil
}

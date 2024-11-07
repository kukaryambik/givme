package cmd

import (
	"fmt"

	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func (opts *CommandOptions) Exec() error {

	img, err := opts.Extract()
	if err != nil {
		return err
	}

	// Get the image config
	logrus.Debugf("Fetching config file for image %s", img.Name)
	cfg, err := img.Config()
	if err != nil {
		return fmt.Errorf("error getting config from image %s: %v", img, err)
	}

	// Prepare the command
	command := opts.PrepareEntrypoint(&cfg.Config)

	// Prepare environment variables
	logrus.Info("Preparing environment variables")
	env, err := opts.PrepareEnvForExec(&cfg.Config)
	if err != nil {
		return err
	}

	// Change the working directory
	if err := unix.Chdir(util.Coalesce(opts.Cwd, cfg.Config.WorkingDir, "/")); err != nil {
		return fmt.Errorf("invalid working directory %q: %v", opts.Cwd, err)
	}

	// Run the command
	return unix.Exec(command[0], command, env)
}

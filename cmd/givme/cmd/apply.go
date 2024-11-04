package cmd

import (
	"fmt"
	"io"
	"strings"
	"syscall"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/kukaryambik/givme/pkg/util"
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

	// Prepare environment variables
	env, err := envars.PrepareEnv(defaultDotEnvFile(), true, opts.OverwriteEnv, cfg.Config.Env)
	if err != nil {
		return nil, err
	}

	// Prepare the command to run in the new rootfs
	var shell []string
	switch {
	case opts.Shell != "":
		w, err := envars.Which(env, opts.Shell)
		if err != nil {
			return nil, err
		}
		shell = []string{w, "-c"}
	case len(cfg.Config.Shell) > 0:
		shell = cfg.Config.Shell
	default:
		w, err := envars.CoalesceWhich(env, "bash", "sh")
		if err != nil {
			return nil, err
		}
		shell = []string{w, "-c"}
	}

	logrus.Info("Image applied")

	opts.Cmd = util.CleanList(opts.Cmd)
	cmd := util.Coalesce(strings.Join(opts.Cmd, " "), shell[0])
	if err := syscall.Exec(shell[0], append(shell, cmd), env); err != nil {
		return nil, err
	}

	return img, nil
}

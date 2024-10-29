package cmd

import (
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
)

// Cleanup removes files and directories in the target directory,
// excluding the paths specified in excludes.
func (opts *CommandOptions) purge() error {
	logrus.Infof("Purging rootfs '%s'", opts.RootFS)

	// Configure ignored paths
	ignoreConf := paths.Ignore(opts.IgnorePaths).ExclFromList(opts.RootFS)
	ignores, err := ignoreConf.AddPaths(opts.Workdir).List()
	if err != nil {
		return err
	}

	if err := paths.Rmrf(opts.RootFS, ignores); err != nil {
		return err
	}

	logrus.Info("Rootfs purged")

	return nil
}

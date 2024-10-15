package cmd

import (
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
)

// Cleanup removes files and directories in the target directory,
// excluding the paths specified in excludes.
func cleanup(opts *CommandOptions) error {
	logrus.Debugf("Starting cleanup...")

	// Configure ignored paths
	ignoreConf := paths.Ignore(opts.IgnorePaths).ExclFromList(opts.RootFS)
	ignores, err := ignoreConf.AddPaths(opts.Workdir).List()
	if err != nil {
		return err
	}

	// List all paths
	var lst []string
	if err := paths.GetList(opts.RootFS, ignores, &lst); err != nil {
		logrus.Errorf("Error listing paths: %v", err)
		return err
	}

	logrus.Debugf("Removing paths: %v", lst)
	if err := paths.Rmrf(lst...); err != nil {
		return err
	}

	logrus.Infoln("Cleanup has completed!")
	return nil
}

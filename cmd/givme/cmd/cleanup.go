package cmd

import (
	"github.com/kukaryambik/givme/pkg/listpaths"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

// Cleanup removes files and directories in the target directory,
// excluding the paths specified in excludes.
func cleanup(conf *CommandOptions) error {
	logrus.Debugf("Starting cleanup...")

	// List all paths
	var paths []string
	if err := listpaths.List(conf.RootFS, conf.RootFS, conf.Exclusions, &paths); err != nil {
		logrus.Errorf("Error listing paths: %v", err)
		return err
	}

	logrus.Debugf("Removing paths: %v", paths)
	if err := util.Rmrf(paths...); err != nil {
		return err
	}

	logrus.Infoln("Cleanup has completed!")
	return nil
}

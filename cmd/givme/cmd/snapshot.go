package cmd

import (
	"fmt"
	"os"

	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/list"
	"github.com/sirupsen/logrus"
)

const (
	defaultSnapshotFile = "snapshot.tar"
	defaultDotenvFile   = ".env"
)

// Snapshot creates a tar archive of the rootfs directory, excluding
// the directories specified in buildExclusions.
func snapshot(conf *CommandOptions) error {
	logrus.Debugf("Starting snapshot creation...")
	logrus.Trace(conf)

	// List all paths
	var paths []string
	if err := list.ListPaths(conf.RootFS, conf.Exclusions, &paths); err != nil {
		return err
	}

	// Check if the file already exists.
	if _, err := os.Stat(conf.TarFile); err == nil {
		logrus.Warnf("File %s already exists", conf.TarFile)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking file %s: %v", conf.TarFile, err)
	} else if os.IsNotExist(err) {
		// Save all environment variables to the file
		logrus.Debugf("Saving environment variables to %s", conf.DotenvFile)
		if err := envars.SaveToFile(os.Environ(), conf.DotenvFile); err != nil {
			return fmt.Errorf("error saving environment variables %s: %v", conf.DotenvFile, err)
		}
		// Create the tar archive
		logrus.Debugf("Creating tar archive: %s", conf.TarFile)
		if err := archiver.Tar(paths, conf.TarFile); err != nil {
			return err
		}
		logrus.Infoln("Snapshot has created!")
	}
	return nil
}

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
func snapshot(rootfs, file, dotenv string, excl []string) error {
	logrus.Debugf("Starting snapshot creation...")
	logrus.Trace(rootfs, file, dotenv, excl)

	// List all paths
	var paths []string
	if err := list.ListPaths(rootfs, excl, &paths); err != nil {
		return err
	}

	// Check if the file already exists.
	if _, err := os.Stat(file); err == nil {
		logrus.Warnf("File %s already exists", file)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking file %s: %v", file, err)
	} else if os.IsNotExist(err) {
		// Save all environment variables to the file
		logrus.Debugf("Saving environment variables to %s", dotenv)
		if err := envars.SaveToFile(os.Environ(), dotenv); err != nil {
			return fmt.Errorf("error saving environment variables %s: %v", dotenv, err)
		}
		// Create the tar archive
		logrus.Debugf("Creating tar archive: %s", file)
		if err := archiver.Tar(paths, file); err != nil {
			return err
		}
		logrus.Infoln("Snapshot has created!")
	}
	return nil
}

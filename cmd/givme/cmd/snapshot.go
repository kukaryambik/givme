package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

var defaultSnapshotName = "snapshot_" + time.Now().Format("20060102150405")

// Snapshot creates a tar archive of the rootfs directory, excluding
// the directories specified in buildExclusions.
func snapshot(opts *CommandOptions) error {
	logrus.Debugf("Starting snapshot creation...")

	// Check if the file already exists.
	if util.IsFileExists(opts.TarFile) {
		logrus.Warnf("File %s already exists", opts.TarFile)
		return nil
	}

	// Save all environment variables to the file
	logrus.Debugf("Saving environment variables to %s", opts.DotenvFile)
	if err := envars.SaveToFile(os.Environ(), opts.DotenvFile); err != nil {
		return fmt.Errorf("error saving environment variables %s: %v", opts.DotenvFile, err)
	}

	// Create the tar archive
	logrus.Debugf("Creating tar archive: %s", opts.TarFile)
	if err := archiver.Tar(opts.RootFS, opts.TarFile, opts.Exclusions); err != nil {
		return err
	}
	logrus.Infof("Snapshot has created!\n\ttarball: %s\n\tdotenv: %s", opts.TarFile, opts.DotenvFile)

	return nil
}

package cmd

import (
	"fmt"
	"os"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
)

var defaultSnapshotFile = "snapshot_" + time.Now().Format("20060102150405") + ".tar"

// Snapshot creates a tar archive of the rootfs directory, excluding
// the directories specified in buildExclusions.
func snapshot(opts *CommandOptions) error {
	logrus.Debugf("Starting snapshot creation...")

	// Check if the file already exists.
	if paths.IsFileExists(opts.TarFile) {
		logrus.Warnf("File %s already exists", opts.TarFile)
		return nil
	}

	// Configure ignored paths
	ignoreConf := paths.Ignore(opts.IgnorePaths).ExclFromList(opts.RootFS)
	ignores, err := ignoreConf.AddPaths(opts.Workdir).List()
	if err != nil {
		return err
	}

	// Create the tar archive of fs
	fsTarBall := opts.TarFile + ".tmp"
	logrus.Debugf("Creating tar archive: %s", fsTarBall)
	if err := archiver.Tar(opts.RootFS, fsTarBall, ignores); err != nil {
		return err
	}
	defer os.Remove(fsTarBall)

	// Create the image
	config := v1.Config{
		Env: os.Environ(),
	}
	config.WorkingDir, err = os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory: %v", err)
	}
	if _, err := image.New(nil, fsTarBall, opts.TarFile, config); err != nil {
		return fmt.Errorf("error creating image: %v", err)
	}

	logrus.Infof("Snapshot has created!")
	fmt.Println(opts.TarFile)

	return nil
}

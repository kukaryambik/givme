package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
)

var defaultSnapshotFile = sync.OnceValue(func() string { return "snapshot_" + time.Now().Format("20060102150405") + ".tar" })

// Snapshot creates a tar archive of the rootfs directory, excluding
// the directories specified in buildExclusions.
func (opts *CommandOptions) snapshot() error {
	logrus.Info("Creating snapshot")

	// Check if the file already exists.
	if paths.FileExists(opts.TarFile) {
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
	tmpTar := filepath.Join(defaultCacheDir(), defaultSnapshotFile())
	logrus.Debugf("Creating tar archive: %s", tmpTar)
	if err := archiver.Tar(opts.RootFS, tmpTar, ignores); err != nil {
		return err
	}
	defer os.Remove(tmpTar)

	// Create the image
	config := v1.Config{
		Env: os.Environ(),
	}
	config.WorkingDir, err = os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory: %v", err)
	}
	if _, err := image.New(nil, tmpTar, opts.TarFile, config); err != nil {
		return fmt.Errorf("error creating image: %v", err)
	}

	fmt.Println(opts.TarFile)
	return nil
}

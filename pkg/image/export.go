package image

import (
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/sirupsen/logrus"
)

// Export exports the image filesystem as a tarball to the given path.
func (img *Image) Export(tar string) error {
	logrus.Debugf("Starting to export filesystem of image %s to %s", img.Name, tar)

	// Create the output tar file
	tarFile, err := os.OpenFile(tar, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Errorf("Error creating file %s: %v", tar, err)
		return fmt.Errorf("error creating file %s: %v", tar, err)
	}
	defer tarFile.Close()

	// Export the image filesystem to the tar file
	logrus.Debugf("Exporting filesystem of image %s to %s", img.Name, tar)
	if err := crane.Export(img.Image, tarFile); err != nil {
		logrus.Errorf("Error exporting image %s to %s: %v", img.Name, tar, err)
		if err := os.Remove(tar); err != nil {
			return fmt.Errorf("error removing image %s: %v", img.Name, err)
		}
		return fmt.Errorf("error exporting image %s: %v", img.Name, err)
	}

	logrus.Debugf("Successfully exported filesystem of image %s to %s", img.Name, tar)
	return nil
}
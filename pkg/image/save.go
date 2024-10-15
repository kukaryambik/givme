package image

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/sirupsen/logrus"
)

// Save saves the full image to a tarball.
func (img *Image) Save(path string) error {

	// Export the entire image as a tarball
	if err := crane.Save(img.Image, img.Name, path); err != nil {
		return fmt.Errorf("error saving image to tar file %s: %v", path, err)
	}

	logrus.Debugf("Image saved as tarball: %s", path)
	return nil
}

package image

import (
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	oci "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sirupsen/logrus"
)

// getImage pulls the image using default Docker authentication.
func getImage(logger *logrus.Logger, image string) (oci.Image, error) {
	logger.Debugf("Pulling image: %s", image)
	authOption := crane.WithAuthFromKeychain(authn.DefaultKeychain)
	img, err := crane.Pull(image, authOption)
	if err != nil {
		logger.Errorf("Error pulling image %s: %v", image, err)
		return nil, fmt.Errorf("error pulling image %s: %v", image, err)
	}
	logger.Debugf("Successfully pulled image: %s", image)
	return img, nil
}

// GetFS pulls an image and exports its filesystem as a tarball to the given path.
func GetFS(logger *logrus.Logger, image, tar string) error {
	logger.Debugf("Starting to export filesystem of image %s to %s", image, tar)

	// Check if the target tar file already exists.
	if _, err := os.Stat(tar); err == nil {
		logger.Warnf("Image %s already downloaded, skipping export", image)
		return nil
	} else if !os.IsNotExist(err) {
		logger.Errorf("Error checking file %s: %v", tar, err)
		return fmt.Errorf("error checking file %s: %v", tar, err)
	}

	// Pull the image
	img, err := getImage(logger, image)
	if err != nil {
		return err
	}

	// Create the output tar file
	tarFile, err := os.OpenFile(tar, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		logger.Errorf("Error creating file %s: %v", tar, err)
		return fmt.Errorf("error creating file %s: %v", tar, err)
	}
	defer tarFile.Close()

	// Export the image filesystem to the tar file
	logger.Debugf("Exporting filesystem of image %s to %s", image, tar)
	if err := crane.Export(img, tarFile); err != nil {
		logger.Errorf("Error exporting image %s to %s: %v", image, tar, err)
		if err := os.Remove(tar); err != nil {
			return fmt.Errorf("error removing image %s: %v", image, err)
		}
		return fmt.Errorf("error exporting image %s: %v", image, err)
	}

	logger.Debugf("Successfully exported filesystem of image %s to %s", image, tar)
	return nil
}

// GetEnv pulls an image and returns its environment variables.
func GetEnv(logger *logrus.Logger, image string) ([]string, error) {
	logger.Debugf("Fetching environment variables for image: %s", image)

	// Pull the image
	img, err := getImage(logger, image)
	if err != nil {
		return nil, err
	}

	// Get the config file of the image
	configFile, err := img.ConfigFile()
	if err != nil {
		logger.Errorf("Error fetching config file for image %s: %v", image, err)
		return nil, fmt.Errorf("error fetching config file for image %s: %v", image, err)
	}

	logger.Debugf("Successfully fetched environment variables for image: %s", image)
	// Return the environment variables from the config file
	return configFile.Config.Env, nil
}

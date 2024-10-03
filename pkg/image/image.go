package image

import (
	"fmt"
	"os"
	"strings"

	"encoding/json"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sirupsen/logrus"
)

// Image represents a pulled container image
type Image struct {
	Image v1.Image
	Name  string
}

// Pull pulls the image using both provided credentials and the default keychain.
func Pull(auth *authn.Basic, image string) (*Image, error) {
	logrus.Debugf("Pulling image: %s", image)

	// Trying to pull the image with anonymous access
	img, err := crane.Pull(image, crane.WithAuth(authn.Anonymous))
	if err == nil {
		logrus.Debugf("Successfully pulled image without credentials: %s", image)
		return &Image{Image: img, Name: image}, nil
	}

	// Checking if the error is an authentication error
	if !isUnauthorizedError(err) {
		logrus.Errorf("Error pulling image %s: %v", image, err)
		return nil, fmt.Errorf("error pulling image %s: %v", image, err)
	}

	// If valid credentials are available, retry with them
	if auth != nil && auth.Username != "" && auth.Password != "" {
		logrus.Debugf("Retrying pulling image with credentials: %s", image)
		basicAuth := authn.FromConfig(authn.AuthConfig{
			Username: auth.Username,
			Password: auth.Password,
		})
		img, err = crane.Pull(image, crane.WithAuth(basicAuth))
		if err != nil {
			logrus.Errorf("Error pulling image with credentials %s: %v", image, err)
			return nil, fmt.Errorf("error pulling image with credentials %s: %v", image, err)
		}
		logrus.Debugf("Successfully pulled image with credentials: %s", image)
		return &Image{Image: img, Name: image}, nil
	}

	// If no valid credentials are available or pulling with them failed
	logrus.Errorf("Error pulling image %s: %v", image, err)
	return nil, fmt.Errorf("error pulling image %s: %v", image, err)
}

// Helper function to check if the error is an authentication error
func isUnauthorizedError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "401 Unauthorized") || strings.Contains(err.Error(), "DENIED")
}

// GetFS pulls an image and exports its filesystem as a tarball to the given path.
func (img *Image) GetFS(tar string) error {
	logrus.Debugf("Starting to export filesystem of image %s to %s", img.Name, tar)

	// Check if the target tar file already exists.
	if _, err := os.Stat(tar); err == nil {
		logrus.Warnf("Image %s already downloaded, skipping export", img.Name)
		return nil
	} else if !os.IsNotExist(err) {
		logrus.Errorf("Error checking file %s: %v", tar, err)
		return fmt.Errorf("error checking file %s: %v", tar, err)
	}

	// Create the output tar file
	tarFile, err := os.OpenFile(tar, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
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

// GetEnv pulls an image and returns its environment variables.
func (img *Image) GetEnv() ([]string, error) {
	logrus.Debugf("Fetching environment variables for image: %s", img.Name)

	// Get the config file of the image
	configFile, err := img.Image.ConfigFile()
	if err != nil {
		logrus.Errorf("Error fetching config file for image %s: %v", img.Name, err)
		return nil, fmt.Errorf("error fetching config file for image %s: %v", img.Name, err)
	}

	logrus.Debugf("Successfully fetched environment variables for image: %s", img.Name)
	// Return the environment variables from the config file
	return configFile.Config.Env, nil
}

// GetConfig pulls an image and returns its config
func (img *Image) GetConfig(file ...string) (v1.ConfigFile, error) {
	logrus.Debugf("Fetching environment variables for image: %s", img.Name)

	var config v1.ConfigFile

	if len(file) > 0 {
		// Check if file already exists.
		if _, err := os.Stat(file[0]); err == nil {
			logrus.Warnf("Config for %s already downloaded. Reading from %s", img.Name, file)

			// Read the file
			jsonData, err := os.ReadFile(file[0])
			if err != nil {
				logrus.Warnf("Error reading file %s: %v", file, err)
			} else {
				// Unmarshal the file
				if err := json.Unmarshal(jsonData, &config); err != nil {
					logrus.Trace(string(jsonData))
					logrus.Warnf("Error unmarshalling file %s: %v", file, err)
				} else {
					return config, nil
				}
			}
		} else if !os.IsNotExist(err) {
			logrus.Warnf("Error checking file %s: %v", file, err)
		}
	}

	// Get the config file of the image
	configPtr, err := img.Image.ConfigFile()
	if err != nil {
		logrus.Errorf("Error fetching config file for image %s: %v", img.Name, err)
		return config, fmt.Errorf("error fetching config file for image %s: %v", img.Name, err)
	}
	config = *configPtr

	logrus.Debugf("Successfully fetched config file for image: %s", img.Name)

	if len(file) > 0 {
		configJson, err := json.Marshal(config)
		if err != nil {
			logrus.Errorf("Error marshalling config file for image %s: %v", img.Name, err)
			return config, err
		}

		// Create the output json file
		fi, err := os.OpenFile(file[0], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			logrus.Errorf("Error creating file %s: %v", file[0], err)
			return config, fmt.Errorf("error creating file %s: %v", file, err)
		}
		defer fi.Close()

		// Write the content to the file
		_, err = fi.Write(configJson)
		if err != nil {
			logrus.Errorf("Error writing to file %s: %v", file, err)
			return config, fmt.Errorf("error writing to file %s: %v", file, err)
		}

		return config, nil
	}

	// Return the environment variables from the config file
	return config, nil
}

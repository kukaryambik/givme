package image

import (
	"encoding/json"
	"fmt"
	"os"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/sirupsen/logrus"
)

// Config pulls an image and returns its config
func (img *Image) Config(file ...string) (v1.ConfigFile, error) {
	logrus.Debugf("Fetching environment variables for image: %s", img.Name)

	var config v1.ConfigFile

	if len(file) > 0 {
		// Check if file already exists.
		if _, err := os.Stat(file[0]); err == nil {
			logrus.Infof("Config for %s already downloaded. Reading from %s", img.Name, file)

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

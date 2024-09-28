package image

import (
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	oci "github.com/google/go-containerregistry/pkg/v1"
)

// GetName parses and returns the image name as a string.
func GetName(image string) (string, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return "", fmt.Errorf("error parsing image reference: %v", err)
	}
	return ref.String(), nil
}

// getImage pulls the image using default Docker authentication.
func getImage(image string) (oci.Image, error) {
	authOption := crane.WithAuthFromKeychain(authn.DefaultKeychain)
	img, err := crane.Pull(image, authOption)
	if err != nil {
		return nil, fmt.Errorf("error pulling image %s: %v", image, err)
	}
	return img, nil
}

// GetFS pulls an image and exports its filesystem as a tarball to the given path.
func GetFS(image, tar string) error {
	// Check if the target tar file already exists.
	if _, err := os.Stat(tar); err == nil {
		fmt.Printf("file %s already exists\n", tar)
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking file %s: %v", tar, err)
	}

	// Pull the image
	img, err := getImage(image)
	if err != nil {
		return err
	}

	// Create the output tar file
	tarFile, err := os.OpenFile(tar, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", tar, err)
	}
	defer tarFile.Close()

	// Export the image filesystem to the tar file
	if err := crane.Export(img, tarFile); err != nil {
		os.Remove(tar)
		return fmt.Errorf("error exporting image %s: %v", image, err)
	}

	return nil
}

// GetEnv pulls an image and returns its environment variables.
func GetEnv(image string) ([]string, error) {
	// Pull the image
	img, err := getImage(image)
	if err != nil {
		return nil, err
	}

	// Get the config file of the image
	configFile, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("error fetching config file for image %s: %v", image, err)
	}

	// Return the environment variables from the config file
	return configFile.Config.Env, nil
}

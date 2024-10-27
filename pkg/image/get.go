package image

import (
	"fmt"
	"runtime"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

type GetConf struct {
	File             string
	Image            string
	RegistryMirror   string
	RegistryPassword string
	RegistryUsername string
	Retry            int
}

var (
	craneLoadFunc = crane.Load
	cranePullFunc = crane.Pull
)

// Load loads the image from a tarball.
var Load = load

func load(path string) (*Image, error) {
	img, err := craneLoadFunc(path)
	if err != nil {
		return nil, fmt.Errorf("error loading image from tar file %s: %v", path, err)
	}

	imgNames, err := GetNamesFromTarball(path)
	if err != nil {
		return nil, err
	}

	image := &Image{Image: img}
	if len(imgNames) > 0 {
		image.Name = imgNames[0]
	}

	return image, nil
}

// Pull pulls the image using both provided credentials and the default keychain.
var Pull = pull

func pull(auth *authn.Basic, image, mirror string) (*Image, error) {
	logrus.Debugf("Pulling image: %s", image)

	// Set the default platform
	platform := v1.Platform{
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
	}

	// Trying to pull the image with anonymous access
	img, err := cranePullFunc(
		withMirror(image, mirror),
		crane.WithAuth(authn.Anonymous),
		crane.WithPlatform(&platform),
	)
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
		img, err = cranePullFunc(
			withMirror(image, mirror),
			crane.WithAuth(basicAuth),
			crane.WithPlatform(&platform),
		)
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

func (conf *GetConf) Get() (*Image, error) {
	if paths.IsFileExists(conf.Image) {
		conf.File = conf.Image
	}

	if !paths.IsFileExists(conf.File) {
		auth := &authn.Basic{
			Username: conf.RegistryUsername,
			Password: conf.RegistryPassword,
		}
		err := util.Retry(conf.Retry, 5*time.Second, func() error {
			img, err := Pull(auth, conf.Image, conf.RegistryMirror)
			if err != nil {
				return err
			}
			return img.Save(conf.File)
		})
		if err != nil {
			return nil, err
		}
		logrus.Infof("Image %s has been saved to %s", conf.Image, conf.File)
	}

	// Load the image
	img, err := Load(conf.File)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Using file %s", conf.File)
	return img, nil
}

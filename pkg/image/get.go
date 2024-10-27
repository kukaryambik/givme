package image

import (
	"fmt"
	"runtime"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
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
	CacheDir         string
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
func (conf *GetConf) Pull() (*Image, error) {
	logrus.Debugf("Pulling image: %s", conf.Image)

	// Set the default platform
	platform := v1.Platform{
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
	}

	nameWithMirror := withMirror(conf.Image, conf.RegistryMirror)
	var image v1.Image

	// Trying to pull the image with anonymous access
	image, err := cranePullFunc(
		nameWithMirror, crane.WithPlatform(&platform), crane.WithAuth(authn.Anonymous),
	)

	switch {
	case err == nil:
		logrus.Debugf("Successfully pulled image: %s", conf.Image)

	// If valid credentials are available, retry with them
	case isUnauthorizedError(err) && conf.RegistryUsername+conf.RegistryPassword != "":
		logrus.Debugf("Retrying pulling image with credentials")
		basicAuth := authn.FromConfig(
			authn.AuthConfig{
				Username: conf.RegistryUsername,
				Password: conf.RegistryPassword,
			},
		)
		if image, err = cranePullFunc(
			nameWithMirror, crane.WithPlatform(&platform), crane.WithAuth(basicAuth),
		); err != nil {
			return nil, fmt.Errorf("error pulling image with credentials %s: %v", image, err)
		}

	default:
		return nil, fmt.Errorf("error pulling image %s: %v", image, err)
	}

	// Set up the cache directory
	blobCache := cache.NewFilesystemCache(conf.CacheDir)
	cachedImage := cache.Image(image, blobCache)

	logrus.Debugf("Successfully pulled image with credentials: %s", image)
	return &Image{Image: cachedImage, Name: conf.Image}, nil
}

func (conf *GetConf) Get() (*Image, error) {
	if paths.IsFileExists(conf.Image) {
		conf.File = conf.Image
	}

	if !paths.IsFileExists(conf.File) {
		err := util.Retry(conf.Retry, 5*time.Second, func() error {
			img, err := conf.Pull()
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

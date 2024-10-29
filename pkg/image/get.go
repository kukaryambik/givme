package image

import (
	"fmt"
	"runtime"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
)

type GetConf struct {
	File             string
	Image            string
	RegistryMirror   string
	RegistryPassword string
	RegistryUsername string
	CacheDir         string
	Update           bool
}

// Load loads the image from a tarball.
var Load = load

func load(path string) (*Image, error) {
	img, err := crane.Load(path)
	if err != nil {
		return nil, fmt.Errorf("error loading image from tar file %s: %v", path, err)
	}

	imgNames, err := GetNamesFromTarball(path)
	if err != nil {
		return nil, err
	}

	image := &Image{Image: img, File: path}
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

	name, err := GetName(conf.Image)
	if err != nil {
		return nil, err
	}
	nameWithMirror, err := withMirror(name, conf.RegistryMirror)
	if err != nil {
		return nil, err
	}

	var image v1.Image

	opts := []remote.Option{
		remote.WithPlatform(platform),
		remote.WithJobs(runtime.NumCPU()),
		remote.WithRetryBackoff(remote.Backoff{
			Duration: 1 * time.Second,
			Factor:   2.0,
			Jitter:   0.15,
			Steps:    5,
			Cap:      10 * time.Second,
		}),
	}

	// Trying to pull the image with anonymous access
	image, err = remote.Image(
		nameWithMirror, append(opts, remote.WithAuth(authn.Anonymous))...,
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
		if image, err = remote.Image(
			nameWithMirror, append(opts, remote.WithAuth(basicAuth))...,
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
	return &Image{Image: cachedImage, Name: name}, nil
}

func (conf *GetConf) Get() (*Image, error) {
	if paths.FileExists(conf.Image) {
		conf.File = conf.Image
	}

	// If the image file exist, just load the image
	if !paths.FileExists(conf.File) || conf.Update {
		i, err := conf.Pull()
		if err != nil {
			return nil, err
		}
		if err := i.Save(conf.File); err != nil {
			return nil, err
		}
	}

	return Load(conf.File)
}

package image

import (
	"fmt"
	"runtime"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func New(ref name.Reference, src, dst string, config v1.Config) (*Image, error) {

	// create a layer new from the tarball
	layer, err := tarball.LayerFromFile(src)
	if err != nil {
		return nil, fmt.Errorf("error reading layer from tarball: %v", err)
	}

	// create an image with the base layer
	image, err := mutate.AppendLayers(empty.Image, layer)
	if err != nil {
		return nil, fmt.Errorf("error appending layers to image: %v", err)
	}

	cfg, err := image.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("error getting config file: %v", err)
	}

	cfg.Config = config
	cfg.Created = v1.Time{Time: time.Now()}
	cfg.Architecture = runtime.GOARCH
	cfg.OS = runtime.GOOS

	// add the config file
	image, err = mutate.ConfigFile(image, cfg)
	if err != nil {
		return nil, fmt.Errorf("error mutating image: %v", err)
	}

	// Save the image
	err = tarball.WriteToFile(dst, ref, image)
	if err != nil {
		return nil, fmt.Errorf("error writing image to tarball: %v", err)
	}

	img := &Image{Image: image}
	if ref != nil {
		img.Name = ref.Name()
	}

	return img, nil
}

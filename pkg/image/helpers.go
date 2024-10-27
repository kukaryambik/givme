package image

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

// Image represents a pulled container image
type Image struct {
	Image v1.Image
	Name  string
}

// GetName returns the fullname of the image
func GetName(i string) (string, error) {
	ref, err := name.ParseReference(i)
	if err != nil {
		return "", fmt.Errorf("error parsing image %s: %v", i, err)
	}

	if ref.Context().RegistryStr() == name.DefaultRegistry {
		suffix := strings.SplitN(ref.Name(), i, 2)[1]
		return i + suffix, nil
	}

	// Return the name
	return ref.Name(), nil
}

func MkImageDir(dir, image string) (string, error) {
	fullName, err := GetName(image)
	if err != nil {
		return "", err
	}
	slugName := util.Slugify(fullName)
	dirName := filepath.Join(dir, slugName)
	if err := os.MkdirAll(dirName, os.ModePerm); err != nil {
		return "", err
	}
	return dirName, nil
}

// GetNamesFromTarball is a helper function to get the image names from a tarball
var GetNamesFromTarball = getNamesFromTarball

func getNamesFromTarball(path string) ([]string, error) {
	opener := func() (io.ReadCloser, error) {
		return os.Open(path)
	}

	manifest, err := tarball.LoadManifest(opener)
	if err != nil {
		return nil, fmt.Errorf("error loading manifest from tarball: %v", err)
	}

	var repoTags []string
	for _, descriptor := range manifest {
		if len(descriptor.RepoTags) > 0 {
			repoTags = append(repoTags, descriptor.RepoTags...)
		}
	}

	if len(repoTags) == 0 {
		logrus.Debugf("No repository tags found in manifest of %s", path)
		return nil, nil
	}

	return repoTags, nil
}

// isUnauthorizedError is a helper function to check if the error is an authentication error
func isUnauthorizedError(err error) bool {
	if err == nil {
		return false
	}
	lowerErr := strings.ToLower(err.Error())
	return strings.Contains(lowerErr, "unauthorized") || strings.Contains(lowerErr, "authentication required")
}

// withMirror updates docker registry of the image to the mirror
func withMirror(img, mirror string) string {
	if mirror == "" {
		return img
	}

	// Parse the reference for the current image
	ref, err := name.ParseReference(img)
	if err != nil {
		logrus.Warnf("Error parsing image %s: %v", img, err)
		return img
	}

	logrus.Debugf("Image successfully parsed: %s. Registry: %s, Repository: %s, Identifier: %s",
		ref.Name(), ref.Context().RegistryStr(), ref.Context().RepositoryStr(), ref.Identifier())

	// If the registry is Docker Hub, replace it with the mirror
	if ref.Context().RegistryStr() == name.DefaultRegistry {
		newReg, err := name.NewRegistry(mirror)
		if err != nil {
			logrus.Warnf("Error parsing registry %s: %v", mirror, err)
			return img
		}

		repo := ref.Context().RepositoryStr()
		ident := ref.Identifier()

		// Return a new image with the updated name (registry mirror)
		if _, ok := ref.(name.Digest); ok {
			return newReg.Repo(repo).Digest(ident).Name()
		} else if _, ok := ref.(name.Tag); ok {
			return newReg.Repo(repo).Tag(ident).Name()
		}
	} else {
		logrus.Debugf("Registry is not Docker Hub: %s, no changes required", ref.Context().RegistryStr())
	}

	// If no changes were made, return the original image
	return img
}

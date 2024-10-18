// image_test.go
package image

import (
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestImage_Config_FileExistsValidJSON(t *testing.T) {
	// Setup
	img := &Image{Name: "test-image"}

	// Mock osStatFunc to simulate file exists
	originalOsStatFunc := osStatFunc
	osStatFunc = func(name string) (os.FileInfo, error) {
		return nil, nil // Simulate that the file exists
	}
	defer func() { osStatFunc = originalOsStatFunc }()

	// Mock osReadFileFunc to return valid JSON
	originalOsReadFileFunc := osReadFileFunc
	osReadFileFunc = func(name string) ([]byte, error) {
		return []byte(`{"architecture": "amd64", "os": "linux"}`), nil
	}
	defer func() { osReadFileFunc = originalOsReadFileFunc }()

	// Execute
	config, err := img.Config("config.json")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "amd64", config.Architecture)
	assert.Equal(t, "linux", config.OS)
}

func TestImage_Config_FileDoesNotExist(t *testing.T) {
	// Setup
	img := &Image{
		Name:  "test-image",
		Image: &fake.FakeImage{},
	}

	// Mock osStatFunc to simulate file does not exist
	originalOsStatFunc := osStatFunc
	osStatFunc = func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	defer func() { osStatFunc = originalOsStatFunc }()

	// Mock img.Image.ConfigFile()
	fakeImage := img.Image.(*fake.FakeImage)
	fakeImage.ConfigFileReturns(&v1.ConfigFile{Architecture: "amd64", OS: "linux"}, nil)

	// Mock osOpenFileFunc to return a temporary file
	originalOsOpenFileFunc := osOpenFileFunc
	osOpenFileFunc = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		// Create a temporary file and return it
		tmpFile, err := os.CreateTemp("", "config-*.json")
		if err != nil {
			return nil, err
		}
		return tmpFile, nil
	}
	defer func() {
		// Clean up the temporary file after the test
		osOpenFileFunc = originalOsOpenFileFunc
	}()

	// Execute
	config, err := img.Config("config.json")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "amd64", config.Architecture)
	assert.Equal(t, "linux", config.OS)
}

func TestImage_Config_ErrorFetchingConfig(t *testing.T) {
	// Setup
	img := &Image{
		Name:  "test-image",
		Image: &fake.FakeImage{},
	}

	// Mock osStatFunc to simulate file does not exist
	originalOsStatFunc := osStatFunc
	osStatFunc = func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	defer func() { osStatFunc = originalOsStatFunc }()

	// Mock img.Image.ConfigFile() to return an error
	fakeImage := img.Image.(*fake.FakeImage)
	fakeImage.ConfigFileReturns(nil, errors.New("error fetching config"))

	// Execute
	config, err := img.Config()

	// Verify
	assert.Error(t, err)
	assert.Equal(t, v1.ConfigFile{}, config)
}

func TestImage_Export_Success(t *testing.T) {
	// Setup
	img := &Image{
		Name:  "test-image",
		Image: &fake.FakeImage{},
	}

	// Mock osOpenFileFunc to prevent actual file creation
	originalOsOpenFileFunc := osOpenFileFunc
	osOpenFileFunc = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		return os.NewFile(0, ""), nil // Return a dummy file
	}
	defer func() { osOpenFileFunc = originalOsOpenFileFunc }()

	// Mock craneExportFunc
	originalCraneExportFunc := craneExportFunc
	craneExportFunc = func(img v1.Image, w io.Writer) error {
		return nil
	}
	defer func() { craneExportFunc = originalCraneExportFunc }()

	// Execute
	err := img.Export("image.tar")

	// Verify
	assert.NoError(t, err)
}

func TestImage_Export_ErrorCreatingFile(t *testing.T) {
	// Setup
	img := &Image{Name: "test-image"}

	// Mock osOpenFileFunc to return an error
	originalOsOpenFileFunc := osOpenFileFunc
	osOpenFileFunc = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		return nil, errors.New("error creating file")
	}
	defer func() { osOpenFileFunc = originalOsOpenFileFunc }()

	// Mock craneExportFunc to ensure it's not called
	originalCraneExportFunc := craneExportFunc
	craneExportFunc = func(img v1.Image, w io.Writer) error {
		t.Fatal("craneExportFunc should not be called when file creation fails")
		return nil
	}
	defer func() { craneExportFunc = originalCraneExportFunc }()

	// Execute
	err := img.Export("image.tar")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error creating file")
}

func TestImage_Export_ErrorDuringExport(t *testing.T) {
	// Setup
	img := &Image{
		Name:  "test-image",
		Image: &fake.FakeImage{},
	}

	// Mock osOpenFileFunc to return a valid temporary file
	originalOsOpenFileFunc := osOpenFileFunc
	osOpenFileFunc = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "export-*.tar")
		if err != nil {
			return nil, err
		}
		return tmpFile, nil
	}
	defer func() { osOpenFileFunc = originalOsOpenFileFunc }()

	// Mock craneExportFunc to return an error
	originalCraneExportFunc := craneExportFunc
	craneExportFunc = func(img v1.Image, w io.Writer) error {
		return errors.New("export error")
	}
	defer func() { craneExportFunc = originalCraneExportFunc }()

	// Execute
	err := img.Export("image.tar")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error exporting image")
}

func TestLoad_Success(t *testing.T) {
	// Setup
	// Mock craneLoadFunc
	originalCraneLoadFunc := craneLoadFunc
	craneLoadFunc = func(path string, opt ...crane.Option) (v1.Image, error) {
		return &fake.FakeImage{}, nil
	}
	defer func() { craneLoadFunc = originalCraneLoadFunc }()

	// Mock GetNamesFromTarball
	originalGetNamesFromTarball := GetNamesFromTarball
	GetNamesFromTarball = func(path string) ([]string, error) {
		return []string{"test-image"}, nil
	}
	defer func() { GetNamesFromTarball = originalGetNamesFromTarball }()

	// Execute
	img, err := Load("image.tar")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "test-image", img.Name)
}

func TestLoad_ErrorLoadingImage(t *testing.T) {
	// Setup
	// Mock craneLoadFunc to return an error
	originalCraneLoadFunc := craneLoadFunc
	craneLoadFunc = func(path string, opt ...crane.Option) (v1.Image, error) {
		return nil, errors.New("load error")
	}
	defer func() { craneLoadFunc = originalCraneLoadFunc }()

	// Execute
	img, err := Load("image.tar")

	// Verify
	assert.Error(t, err)
	assert.Nil(t, img)
	assert.Contains(t, err.Error(), "error loading image from tar file")
}

func TestPull_SuccessAnonymous(t *testing.T) {
	// Setup
	imageName := "test-image"
	auth := &authn.Basic{}
	mirror := ""

	// Mock cranePullFunc
	originalCranePullFunc := cranePullFunc
	cranePullFunc = func(ref string, opt ...crane.Option) (v1.Image, error) {
		return &fake.FakeImage{}, nil
	}
	defer func() { cranePullFunc = originalCranePullFunc }()

	// Execute
	img, err := Pull(auth, imageName, mirror)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, imageName, img.Name)
}

func TestPull_UnauthorizedRetriesWithCredentials(t *testing.T) {
	// Setup
	imageName := "test-image"
	auth := &authn.Basic{
		Username: "user",
		Password: "pass",
	}
	mirror := ""

	// Mock cranePullFunc
	originalCranePullFunc := cranePullFunc
	callCount := 0
	cranePullFunc = func(ref string, opt ...crane.Option) (v1.Image, error) {
		callCount++
		if callCount == 1 {
			// Return an error that includes "unauthorized" to trigger retry logic
			return nil, fmt.Errorf("unauthorized: authentication required")
		}
		return &fake.FakeImage{}, nil
	}
	defer func() { cranePullFunc = originalCranePullFunc }()

	// Execute
	img, err := Pull(auth, imageName, mirror)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, img)
	assert.Equal(t, imageName, img.Name)
	assert.Equal(t, 2, callCount)
}

func TestPull_UnauthorizedNoCredentials(t *testing.T) {
	// Setup
	imageName := "test-image"
	auth := &authn.Basic{}
	mirror := ""

	// Mock cranePullFunc
	originalCranePullFunc := cranePullFunc
	cranePullFunc = func(ref string, opt ...crane.Option) (v1.Image, error) {
		return nil, errors.New("unauthorized: authentication required")
	}
	defer func() { cranePullFunc = originalCranePullFunc }()

	// Execute
	img, err := Pull(auth, imageName, mirror)

	// Verify
	assert.Error(t, err)
	assert.Nil(t, img)
	assert.Contains(t, err.Error(), "error pulling image")
}

func TestImage_Save_Success(t *testing.T) {
	// Setup
	img := &Image{
		Name:  "test-image",
		Image: &fake.FakeImage{},
	}

	// Mock craneSaveFunc
	originalCraneSaveFunc := craneSaveFunc
	craneSaveFunc = func(img v1.Image, ref string, path string) error {
		return nil
	}
	defer func() { craneSaveFunc = originalCraneSaveFunc }()

	// Execute
	err := img.Save("image.tar")

	// Verify
	assert.NoError(t, err)
}

func TestImage_Save_Error(t *testing.T) {
	// Setup
	img := &Image{
		Name:  "test-image",
		Image: &fake.FakeImage{},
	}

	// Mock craneSaveFunc to return an error
	originalCraneSaveFunc := craneSaveFunc
	craneSaveFunc = func(img v1.Image, ref string, path string) error {
		return errors.New("save error")
	}
	defer func() { craneSaveFunc = originalCraneSaveFunc }()

	// Execute
	err := img.Save("image.tar")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error saving image to tar file")
}

func TestGetConf_Get_FileExists(t *testing.T) {
	// Setup
	conf := &GetConf{
		File:  "image.tar",
		Image: "test-image",
		Retry: 1,
	}

	// Mock paths.IsFileExists to return true
	originalIsFileExists := paths.IsFileExists
	paths.IsFileExists = func(path string) bool {
		return true
	}
	defer func() { paths.IsFileExists = originalIsFileExists }()

	// Mock Load function
	originalLoad := Load
	Load = func(path string) (*Image, error) {
		return &Image{Name: "test-image"}, nil
	}
	defer func() { Load = originalLoad }()

	// Execute
	img, err := conf.Get()

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "test-image", img.Name)
}

func TestGetConf_Get_FileDoesNotExist(t *testing.T) {
	// Setup
	conf := &GetConf{
		File:  "image.tar",
		Image: "test-image",
		Retry: 1,
	}

	// Mock paths.IsFileExists to return false
	originalIsFileExists := paths.IsFileExists
	paths.IsFileExists = func(path string) bool {
		return false
	}
	defer func() { paths.IsFileExists = originalIsFileExists }()

	// Mock Pull function
	originalPull := Pull
	Pull = func(auth *authn.Basic, image, mirror string) (*Image, error) {
		return &Image{Name: image, Image: &fake.FakeImage{}}, nil
	}
	defer func() { Pull = originalPull }()

	// Mock craneSaveFunc to prevent actual file operations
	originalCraneSaveFunc := craneSaveFunc
	craneSaveFunc = func(img v1.Image, ref string, path string) error {
		return nil
	}
	defer func() { craneSaveFunc = originalCraneSaveFunc }()

	// Mock Load function
	originalLoad := Load
	Load = func(path string) (*Image, error) {
		return &Image{Name: "test-image"}, nil
	}
	defer func() { Load = originalLoad }()

	// Mock util.Retry
	originalRetry := util.Retry
	util.Retry = func(attempts int, sleep time.Duration, fn func() error) error {
		return fn()
	}
	defer func() { util.Retry = originalRetry }()

	// Execute
	img, err := conf.Get()

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, "test-image", img.Name)
}

func TestGetConf_Get_ErrorPullingImage(t *testing.T) {
	// Setup
	conf := &GetConf{
		File:  "image.tar",
		Image: "test-image",
		Retry: 2,
	}

	// Mock paths.IsFileExists to return false
	originalIsFileExists := paths.IsFileExists
	paths.IsFileExists = func(path string) bool {
		return false
	}
	defer func() { paths.IsFileExists = originalIsFileExists }()

	// Mock Pull to return an error
	originalPull := Pull
	Pull = func(auth *authn.Basic, image, mirror string) (*Image, error) {
		return nil, errors.New("pull error")
	}
	defer func() { Pull = originalPull }()

	// Mock util.Retry
	originalRetry := util.Retry
	util.Retry = func(attempts int, sleep time.Duration, fn func() error) error {
		var lastErr error
		for i := 0; i < attempts; i++ {
			lastErr = fn()
			if lastErr == nil {
				return nil
			}
			time.Sleep(sleep)
		}
		return lastErr
	}
	defer func() { util.Retry = originalRetry }()

	// Execute
	img, err := conf.Get()

	// Verify
	assert.Error(t, err)
	assert.Nil(t, img)
	assert.Contains(t, err.Error(), "pull error")
}

func TestGetName_ValidReference(t *testing.T) {
	imageName := "docker.io/library/alpine:latest"

	name, err := GetName(imageName)

	assert.NoError(t, err)
	assert.Equal(t, "library/alpine:latest", name)
}

func TestGetName_InvalidReference(t *testing.T) {
	imageName := "invalid@@@image"

	name, err := GetName(imageName)

	assert.Error(t, err)
	assert.Empty(t, name)
}

func TestMkImageDir_Success(t *testing.T) {
	// Setup
	tempDir := os.TempDir()
	image := "docker.io/library/alpine:latest"

	// Execute
	dirPath, err := MkImageDir(tempDir, image)

	// Verify
	assert.NoError(t, err)
	assert.DirExists(t, dirPath)

	// Cleanup
	os.RemoveAll(dirPath)
}

func TestMkImageDir_ErrorGetName(t *testing.T) {
	// Setup
	tempDir := os.TempDir()
	image := "invalid@@@image"

	// Execute
	dirPath, err := MkImageDir(tempDir, image)

	// Verify
	assert.Error(t, err)
	assert.Empty(t, dirPath)
}

func TestWithMirror_ReplaceDockerHub(t *testing.T) {
	image := "docker.io/library/alpine:latest"
	mirror := "mirror.registry.io"

	result := withMirror(image, mirror)

	expected := "mirror.registry.io/library/alpine:latest"
	assert.Equal(t, expected, result)
}

func TestWithMirror_NonDockerHubRegistry(t *testing.T) {
	image := "gcr.io/project/image:tag"
	mirror := "mirror.registry.io"

	result := withMirror(image, mirror)

	assert.Equal(t, image, result)
}

func TestWithMirror_NoMirrorProvided(t *testing.T) {
	image := "docker.io/library/alpine:latest"
	mirror := ""

	result := withMirror(image, mirror)

	assert.Equal(t, image, result)
}

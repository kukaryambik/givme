package archiver

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/util"
	"github.com/mholt/archiver/v4"
	"github.com/sirupsen/logrus"
)

func Tar(src []string, dst string) error {
	filesMap := make(map[string]string)
	for _, srcPath := range src {
		// Map source file paths, trimming leading slashes
		filesMap[srcPath] = strings.TrimLeft(srcPath, "/")
	}

	// Collect files from disk for archiving
	files, err := archiver.FilesFromDisk(nil, filesMap)
	if err != nil {
		logrus.Errorf("Error collecting files from disk: %v", err)
		return err
	}

	// Create the destination archive file
	out, err := os.Create(dst)
	if err != nil {
		logrus.Errorf("Error creating destination file %s: %v", dst, err)
		return err
	}
	defer out.Close()

	// Initialize the tar format
	tar := archiver.Tar{ContinueOnError: true}

	// Archive the files to the output
	err = tar.Archive(context.Background(), out, files)
	if err != nil {
		logrus.Errorf("Error archiving files: %v", err)
		return err
	}

	logrus.Debugf("Successfully created tar archive: %s", dst)
	return nil
}

func Untar(src, dst string, excl []string) error {
	// Open the source archive for reading
	input, err := os.Open(src)
	if err != nil {
		logrus.Errorf("Error opening source archive %s: %v", src, err)
		return err
	}
	defer input.Close()

	// Initialize the tar format
	tar := archiver.Tar{ContinueOnError: true}

	// Define handler for extracting files from the archive
	handler := func(ctx context.Context, file archiver.File) error {
		targetPath := filepath.Join(dst, file.NameInArchive)

		// Check if the path should be excluded using util.IsPathFrom
		shouldExclude, err := util.IsPathFrom(targetPath, excl)
		if err != nil {
			return err
		}

		// Skip excluded paths
		if shouldExclude {
			logrus.Debugf("Skipping excluded path: %s", targetPath)
			return nil
		}

		// Handle symbolic links
		if file.LinkTarget != "" {
			logrus.Debugf("Creating symbolic link for %s -> %s", targetPath, file.LinkTarget)
			if err := os.RemoveAll(targetPath); err != nil {
				logrus.Errorf("Error removing existing file %s: %v", targetPath, err)
				return err
			}
			if err := os.Symlink(file.LinkTarget, targetPath); err != nil {
				logrus.Errorf("Error creating symbolic link %s: %v", targetPath, err)
				return err
			}
			return nil
		}

		// Create directories if needed
		if file.IsDir() {
			logrus.Debugf("Creating directory: %s", targetPath)
			if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
				logrus.Errorf("Error creating directory %s: %v", targetPath, err)
				return err
			}
			return os.Chmod(targetPath, file.Mode())
		}

		// Ensure parent directories exist
		if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
			logrus.Errorf("Error creating parent directories for %s: %v", targetPath, err)
			return err
		}

		// Open the file for writing, truncating it if it already exists
		outFile, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			logrus.Errorf("Error opening file %s for writing: %v", targetPath, err)
			return nil // Continue with other files
		}
		defer outFile.Close()

		fileReader, err := file.Open()
		if err != nil {
			logrus.Errorf("Error opening archive file %s: %v", file.NameInArchive, err)
			return err
		}
		defer fileReader.Close()

		// Copy file data from the archive to the target file
		_, err = io.Copy(outFile, fileReader)
		if err != nil {
			logrus.Errorf("Error writing to file %s: %v", targetPath, err)
			return nil // Continue with other files
		}

		logrus.Debugf("Extracted file: %s", targetPath)
		return nil
	}

	// Extract files from the tar archive
	err = tar.Extract(context.Background(), input, nil, handler)
	if err != nil {
		logrus.Errorf("Error extracting archive: %v", err)
		return err
	}

	logrus.Debugf("Successfully extracted archive: %s", src)
	return nil
}

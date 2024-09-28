package archiver

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/util"
	"github.com/mholt/archiver/v4"
)

func Tar(sources []string, destination string) error {
	filesMap := make(map[string]string)
	for _, src := range sources {
		// Map source file paths, trimming leading slashes
		filesMap[src] = strings.TrimLeft(src, "/")
	}

	// Collect files from disk for archiving
	files, err := archiver.FilesFromDisk(nil, filesMap)
	if err != nil {
		return err
	}

	// Create the destination archive file
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	// Initialize the tar format
	tar := archiver.Tar{ContinueOnError: true}

	// Archive the files to the output
	err = tar.Archive(context.Background(), out, files)
	if err != nil {
		return err
	}

	return nil
}

func Untar(source, destination string, exclude []string) error {
	// Open the source archive for reading
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	// Initialize the tar format
	tar := archiver.Tar{ContinueOnError: true}

	// Define handler for extracting files from the archive
	handler := func(ctx context.Context, file archiver.File) error {
		targetPath := filepath.Join(destination, file.NameInArchive)

		// Skip excluded paths
		if util.IsPathFrom(targetPath, exclude) {
			return nil
		}

		// Handle symbolic links
		if file.LinkTarget != "" {
			if err := os.Symlink(file.LinkTarget, targetPath); err != nil {
				return err
			}
			return nil
		}

		// Create directories if needed
		if file.IsDir() {
			if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
				return err
			}
			// Set directory permissions
			return os.Chmod(targetPath, file.Mode())
		}

		// Ensure parent directories exist for the file
		if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
			return err
		}

		// Create the file and write its content
		outFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer outFile.Close()

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		// Copy file data from the archive to the target file
		_, err = io.Copy(outFile, fileReader)
		if err != nil {
			return err
		}

		// Set file permissions
		return os.Chmod(targetPath, file.Mode())
	}

	// Extract files from the tar archive
	err = tar.Extract(context.Background(), input, nil, handler)
	if err != nil {
		return err
	}

	return nil
}

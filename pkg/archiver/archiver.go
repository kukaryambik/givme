package archiver

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/kukaryambik/givme/pkg/util"
	"github.com/mholt/archiver/v4"
)

func Tar(sources []string, destination string) error {
	filesMap := make(map[string]string)
	for _, src := range sources {
		dst, err := filepath.Rel("/", src)
		if err != nil {
			return err
		}
		filesMap[src] = dst
	}

	// Map the files from disk to their paths in the archive
	files, err := archiver.FilesFromDisk(nil, filesMap)
	if err != nil {
		return err
	}

	// Create the output file
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	// Create the archive format
	tar := archiver.Tar{ContinueOnError: true}

	// Create the archive
	err = tar.Archive(context.Background(), out, files)
	if err != nil {
		return err
	}

	return nil
}

func Untar(source, destination string, exclude []string) error {
	// Open the source archive file
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	// Create the tar format
	tar := archiver.Tar{ContinueOnError: true}
	// Handler for files inside the archive
	handler := func(ctx context.Context, file archiver.File) error {
		// Full path for saving the file
		targetPath := filepath.Join(destination, file.NameInArchive)

		// Check for excluded paths
		if util.IsPathFrom(targetPath, exclude) {
			return nil
		}

		// Check if the file is a symbolic link
		if file.LinkTarget != "" {
			// Create a symbolic link
			if err := os.Symlink(file.LinkTarget, targetPath); err != nil {
				return err
			}
			return nil
		}

		// If it's a directory, create it
		if file.IsDir() {
			if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
				return err
			}
			// Restore directory permissions
			if err := os.Chmod(targetPath, file.Mode()); err != nil {
				return err
			}
			return nil
		}

		// If it's a file, first create the directory for it
		if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
			return err
		}

		// Create the file for writing
		outFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer outFile.Close()

		// Open the file from the archive
		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		// Copy the file contents from the archive to the file on disk
		_, err = io.Copy(outFile, fileReader)
		if err != nil {
			return err
		}

		// Restore file permissions
		if err := os.Chmod(targetPath, file.Mode()); err != nil {
			return err
		}

		return nil
	}

	// Extract the tar archive
	err = tar.Extract(context.Background(), input, nil, handler)
	if err != nil {
		return err
	}

	return nil
}

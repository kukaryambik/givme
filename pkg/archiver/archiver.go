package archiver

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func Tar(src, dst string, excl []string) error {
	// Open the destination file for writing the archive
	outFile, err := os.Create(dst)
	if err != nil {
		logrus.Errorf("Error creating archive file %s: %v", dst, err)
		return err
	}
	defer outFile.Close()

	// Create a new tar.Writer
	tarWriter := tar.NewWriter(outFile)
	defer tarWriter.Close()

	// Walk through the source directory
	err = filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			logrus.Errorf("Error accessing file %s: %v", file, err)
			return err
		}

		// Determine the relative path for storage in the archive
		relPath, err := filepath.Rel(src, file)
		if err != nil {
			logrus.Errorf("Error calculating relative path for %s: %v", file, err)
			return err
		}

		// Check if the path should be excluded
		shouldExclude, err := util.IsPathFrom(file, excl)
		if err != nil {
			return err
		}
		if shouldExclude {
			logrus.Tracef("Skipping excluded file or directory: %s", file)
			if fi.IsDir() {
				return filepath.SkipDir // Skip this directory and its contents
			}
			return nil // Skip this file
		}

		// Create the tar header
		var linkTarget string
		if fi.Mode()&os.ModeSymlink != 0 {
			// This is a symbolic link, get the link target
			linkTarget, err = os.Readlink(file)
			if err != nil {
				logrus.Errorf("Error reading symbolic link %s: %v", file, err)
				return err
			}
		}

		hdr, err := tar.FileInfoHeader(fi, linkTarget)
		if err != nil {
			logrus.Errorf("Error creating tar header for %s: %v", file, err)
			return err
		}

		// Adjust the file name in the header to be relative to the source directory
		hdr.Name = relPath

		// Write the header to the archive
		if err := tarWriter.WriteHeader(hdr); err != nil {
			logrus.Errorf("Error writing header for %s: %v", file, err)
			return err
		}

		// If it's not a regular file, there's no data to write
		if !fi.Mode().IsRegular() {
			return nil
		}

		// Open the file for reading
		f, err := os.Open(file)
		if err != nil {
			logrus.Errorf("Error opening file %s: %v", file, err)
			return err
		}
		defer f.Close()

		// Copy the file data into the archive
		if _, err := io.Copy(tarWriter, f); err != nil {
			logrus.Errorf("Error writing file %s to archive: %v", file, err)
			return err
		}

		logrus.Tracef("Added file to archive: %s", file)
		return nil
	})
	if err != nil {
		logrus.Errorf("Error walking through source directory %s: %v", src, err)
		return err
	}

	logrus.Debugf("Archive successfully created: %s", dst)
	return nil
}

func Untar(src, dst string, excl []string) error {
	// First, process directories
	if err := processDirs(src, dst, excl); err != nil {
		return err
	}

	// Then, process files
	if err := processFiles(src, dst, excl); err != nil {
		return err
	}

	// Finally, process links
	if err := processLinks(src, dst, excl); err != nil {
		return err
	}

	logrus.Debugf("Archive successfully unpacked: %s", src)
	return nil
}

func processDirs(src, dst string, excl []string) error {
	// Open the source archive for reading
	input, err := os.Open(src)
	if err != nil {
		logrus.Errorf("Error opening archive %s: %v", src, err)
		return err
	}
	defer input.Close()

	tarReader := tar.NewReader(input)

	// Create a structure to store the directory name and its permissions
	type dirEntry struct {
		Name string
		Mode os.FileMode
	}

	var dirEntries []dirEntry

	// Read entries and collect directories
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			// End of archive
			break
		}
		if err != nil {
			logrus.Errorf("Error reading archive entry: %v", err)
			return err
		}

		if hdr.Typeflag == tar.TypeDir {
			dirEntries = append(dirEntries, dirEntry{
				Name: hdr.Name,
				Mode: hdr.FileInfo().Mode(),
			})
		}
	}

	// Sort directories from root to deeper levels
	sort.Slice(dirEntries, func(i, j int) bool {
		iDepth := strings.Count(dirEntries[i].Name, string(os.PathSeparator))
		jDepth := strings.Count(dirEntries[j].Name, string(os.PathSeparator))
		return iDepth < jDepth
	})

	// Create directories with the correct permissions
	for _, dir := range dirEntries {
		targetPath := filepath.Join(dst, dir.Name)

		// Check if the path should be excluded
		shouldExclude, err := util.IsPathFrom(targetPath, excl)
		if err != nil {
			return err
		}
		if shouldExclude {
			logrus.Tracef("Skipping excluded directory: %s", targetPath)
			continue
		}

		// Create the directory with permissions from the archive
		if err := os.MkdirAll(targetPath, dir.Mode); err != nil {
			logrus.Errorf("Error creating directory %s: %v", targetPath, err)
			return err
		}

		// Set exact permissions (in case os.MkdirAll changed them)
		if err := os.Chmod(targetPath, dir.Mode); err != nil {
			logrus.Errorf("Error setting permissions for directory %s: %v", targetPath, err)
			return err
		}

		logrus.Tracef("Created directory: %s with permissions %v", targetPath, dir.Mode)
	}

	return nil
}

func processFiles(src, dst string, excl []string) error {
	// Open the source archive for reading
	input, err := os.Open(src)
	if err != nil {
		logrus.Errorf("Error opening archive %s: %v", src, err)
		return err
	}
	defer input.Close()

	tarReader := tar.NewReader(input)

	// Read entries and process files
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			// End of archive
			break
		}
		if err != nil {
			logrus.Errorf("Error reading archive entry: %v", err)
			return err
		}

		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			// Skip non-regular files
			continue
		}

		targetPath := filepath.Join(dst, hdr.Name)

		// Check if the path should be excluded
		shouldExclude, err := util.IsPathFrom(targetPath, excl)
		if err != nil {
			return err
		}
		if shouldExclude {
			logrus.Tracef("Skipping excluded file: %s", targetPath)
			// Skip the file data
			if _, err := io.Copy(io.Discard, tarReader); err != nil {
				return err
			}
			continue
		}

		// Create the file
		outFile, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode())
		if err != nil {
			logrus.Errorf("Error opening file %s for writing: %v", targetPath, err)
			return err
		}

		// Copy the file data
		if _, err := io.Copy(outFile, tarReader); err != nil {
			logrus.Errorf("Error writing file %s: %v", targetPath, err)
			outFile.Close()
			return err
		}
		outFile.Close()

		// Restore the file's modification time
		if err := os.Chtimes(targetPath, hdr.AccessTime, hdr.ModTime); err != nil {
			logrus.Warnf("Error setting times for file %s: %v", targetPath, err)
		}

		logrus.Tracef("Extracted file: %s", targetPath)
	}

	return nil
}

func processLinks(src, dst string, excl []string) error {
	// Open the source archive for reading
	input, err := os.Open(src)
	if err != nil {
		logrus.Errorf("Error opening archive %s: %v", src, err)
		return err
	}
	defer input.Close()

	tarReader := tar.NewReader(input)

	// Read entries and process links
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			// End of archive
			break
		}
		if err != nil {
			logrus.Errorf("Error reading archive entry: %v", err)
			return err
		}

		if hdr.Typeflag != tar.TypeSymlink && hdr.Typeflag != tar.TypeLink {
			// Skip non-links
			continue
		}

		targetPath := filepath.Join(dst, hdr.Name)

		// Check if the path should be excluded
		shouldExclude, err := util.IsPathFrom(targetPath, excl)
		if err != nil {
			return err
		}
		if shouldExclude {
			logrus.Tracef("Skipping excluded link: %s", targetPath)
			continue
		}

		linkTarget := hdr.Linkname

		switch hdr.Typeflag {
		case tar.TypeSymlink:
			// Create a symbolic link
			logrus.Tracef("Creating symbolic link: %s -> %s", targetPath, linkTarget)
			if err := os.RemoveAll(targetPath); err != nil {
				logrus.Errorf("Error removing existing file %s: %v", targetPath, err)
				return err
			}
			if err := os.Symlink(linkTarget, targetPath); err != nil {
				logrus.Errorf("Error creating symbolic link %s: %v", targetPath, err)
				return err
			}
		case tar.TypeLink:
			// Create a hard link
			linkTargetPath := filepath.Join(dst, linkTarget)
			logrus.Tracef("Creating hard link: %s -> %s", targetPath, linkTargetPath)
			if err := os.RemoveAll(targetPath); err != nil {
				logrus.Errorf("Error removing existing file %s: %v", targetPath, err)
				return err
			}
			if err := os.Link(linkTargetPath, targetPath); err != nil {
				logrus.Errorf("Error creating hard link %s: %v", targetPath, err)
				return err
			}
		default:
			logrus.Warnf("Unknown link type for %s, creating symbolic link by default", targetPath)
			if err := os.RemoveAll(targetPath); err != nil {
				logrus.Errorf("Error removing existing file %s: %v", targetPath, err)
				return err
			}
			if err := os.Symlink(linkTarget, targetPath); err != nil {
				logrus.Errorf("Error creating symbolic link %s: %v", targetPath, err)
				return err
			}
		}

		logrus.Tracef("Created link: %s -> %s", targetPath, linkTarget)
	}

	return nil
}

package archiver

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
)

var Chown bool = os.Getuid() == 0

// restorePerm restores the permissions of a file or directory
func restorePerm(path string, info *tar.Header) {
	if !paths.IsFileExists(path) {
		return
	}

	if err := os.Chmod(path, info.FileInfo().Mode()); err != nil {
		logrus.Warnf("Error setting permissions for %s: %v", path, err)
	}

	if err := os.Chtimes(path, info.AccessTime, info.ModTime); err != nil {
		logrus.Warnf("Error setting times for %s: %v", path, err)
	}

	if Chown {
		if err := os.Chown(path, info.Uid, info.Gid); err != nil {
			logrus.Warnf("Error setting owner for %s: %v", path, err)
		}
	}
}

// Untar extracts a tar archive from `src` to `dst`, excluding any paths specified in `excl`.
func Untar(src, dst string, excl []string) error {
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("error getting absolute path for %s: %v", src, err)
	}

	absDst, err := filepath.Abs(dst)
	if err != nil {
		return fmt.Errorf("error getting absolute path for %s: %v", dst, err)
	}

	absExcl, err := paths.AbsAll(excl)
	if err != nil {
		return fmt.Errorf("failed to convert exclusion list to absolute paths: %v", err)
	}

	// First, process directories
	if err := processDirs(absSrc, absDst, absExcl); err != nil {
		return err
	}

	// Then, process files and special files
	if err := processFiles(absSrc, absDst, absExcl); err != nil {
		return err
	}

	// Finally, process links
	if err := processLinks(absSrc, absDst, absExcl); err != nil {
		return err
	}

	logrus.Debugf("Archive successfully unpacked: %s", src)
	return nil
}

// processDirs processes directories from the archive.
func processDirs(src, dst string, excl []string) error {
	// Open the source archive for reading
	input, err := os.Open(src)
	if err != nil {
		logrus.Errorf("Error opening archive %s: %v", src, err)
		return err
	}
	defer input.Close()

	tarReader := tar.NewReader(input)

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
			targetPath := filepath.Join(dst, hdr.Name)

			// Check if the path should be excluded
			if paths.IsPathFrom(targetPath, excl) {
				logrus.Tracef("Skipping excluded path: %s", hdr.Name)
				continue
			}

			// Create the directory with permissions from the archive
			if err := os.MkdirAll(targetPath, hdr.FileInfo().Mode()); err != nil {
				return fmt.Errorf("error creating directory %s: %v", targetPath, err)
			}

			restorePerm(targetPath, hdr)

			logrus.Tracef("Created directory: %s with permissions %v", targetPath, hdr.Mode)
		}
	}

	return nil
}

// processFiles processes regular files and special files like FIFOs.
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

		targetPath := filepath.Join(dst, hdr.Name)

		// Check if the path should be excluded
		if paths.IsPathFrom(targetPath, excl) {
			logrus.Tracef("Skipping excluded path: %s", hdr.Name)
			// Skip the file data if it's a regular file
			if hdr.Typeflag == tar.TypeReg {
				if _, err := io.Copy(io.Discard, tarReader); err != nil {
					return fmt.Errorf("error skipping file %s: %v", targetPath, err)
				}
			}
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeReg:
			// Handle regular files
			outFile, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("error creating file %s: %v", targetPath, err)
			}

			// Copy the file data
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("error writing file %s: %v", targetPath, err)
			}
			outFile.Close()

			restorePerm(targetPath, hdr)

			logrus.Tracef("Extracted file: %s", targetPath)
		case tar.TypeFifo:
			// Handle FIFOs
			if err := processSpecialFiles(hdr, targetPath); err != nil {
				return err
			}
		default:
			// Skip other types for now
			continue
		}
	}

	return nil
}

// processLinks processes symbolic and hard links from the archive.
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
		if paths.IsPathFrom(targetPath, excl) {
			logrus.Tracef("Skipping excluded path: %s", hdr.Name)
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeSymlink:
			// Create a symbolic link
			logrus.Tracef("Creating symbolic link: %s -> %s", targetPath, hdr.Linkname)
			if err := os.RemoveAll(targetPath); err != nil {
				return fmt.Errorf("error removing existing file %s: %v", targetPath, err)
			}
			if err := os.Symlink(hdr.Linkname, targetPath); err != nil {
				return fmt.Errorf("error creating symbolic link %s: %v", targetPath, err)
			}
		case tar.TypeLink:
			// Create a hard link
			linkTargetPath := filepath.Join(dst, hdr.Linkname)
			logrus.Tracef("Creating hard link: %s -> %s", targetPath, linkTargetPath)
			if err := os.RemoveAll(targetPath); err != nil {
				return fmt.Errorf("error removing existing file %s: %v", targetPath, err)
			}
			if err := os.Link(linkTargetPath, targetPath); err != nil {
				return fmt.Errorf("error creating hard link %s: %v", targetPath, err)
			}
		}

		logrus.Tracef("Created link: %s -> %s", targetPath, hdr.Linkname)
	}

	return nil
}

// processSpecialFiles handles special files like FIFOs during extraction.
func processSpecialFiles(hdr *tar.Header, targetPath string) error {
	switch hdr.Typeflag {
	case tar.TypeFifo:
		err := syscall.Mkfifo(targetPath, uint32(hdr.FileInfo().Mode()))
		if err != nil {
			return fmt.Errorf("error creating FIFO %s: %v", targetPath, err)
		}
		restorePerm(targetPath, hdr)
		logrus.Tracef("Created FIFO: %s", targetPath)
	default:
		logrus.Warnf("Unsupported special file type: %v", hdr.Typeflag)
	}
	return nil
}

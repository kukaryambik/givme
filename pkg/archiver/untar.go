// Package archiver provides functionality for extracting tar archives.
// It supports excluding specific paths and handles various types of archive entries
// including regular files, directories, hard links, and symbolic links.
package archiver

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// Chown determines whether to change file ownership during extraction.
// It's set to true if the current user is root (UID 0), false otherwise.
var Chown bool = os.Getuid() == 0

// Untar extracts a tar archive from src to dst, excluding any paths specified in excl.
// It processes the archive in two phases: first collecting directory entries,
// then extracting files in parallel for better performance.
//
// Parameters:
//   - src: io.Reader containing the tar archive data
//   - dst: destination directory path where files will be extracted
//   - excl: slice of paths to exclude from extraction
//
// Returns:
//   - error: nil if successful, otherwise describes the failure
func Untar(src io.Reader, dst string, excl []string) error {
	logrus.Debugf("Unpacking tar archive to %s", dst)
	
	// Convert destination path to absolute path for consistency
	absDst, err := filepath.Abs(dst)
	if err != nil {
		return fmt.Errorf("error getting absolute path for %s: %v", dst, err)
	}
	
	// Convert exclusion list to absolute paths
	absExcl, err := paths.AbsAll(excl)
	if err != nil {
		return fmt.Errorf("failed to convert exclusion list to absolute paths: %v", err)
	}

	tr := tar.NewReader(src)
	hdrs := make(map[string]tar.Header) // Store headers for later processing
	var dirs []string                   // Collect directories to create

	// Phase 1: Read all entries and collect directories
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// End of archive reached
			break
		}
		if err != nil {
			logrus.Errorf("Error reading archive entry: %v", err)
			return err
		}

		targetPath := filepath.Join(absDst, hdr.Name)

		// Check if the path should be excluded
		if paths.PathFrom(targetPath, absExcl) {
			logrus.Tracef("Skipping excluded path: %s", hdr.Name)
			// Skip the file data if it's a regular file
			if hdr.Typeflag == tar.TypeReg {
				if _, err := io.Copy(io.Discard, tr); err != nil {
					return fmt.Errorf("error skipping file %s: %v", targetPath, err)
				}
			}
			continue
		}

		// Create directory structure as we encounter directories
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(targetPath, hdr.FileInfo().Mode()); err != nil {
				return fmt.Errorf("error creating directory %s: %v", targetPath, err)
			}
			logrus.Tracef("Created directory: %s", targetPath)
			dirs = append(dirs, targetPath)
		} else {
			// For non-directory entries, ensure parent directory exists
			parentDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("error creating parent directory %s: %v", parentDir, err)
			}
			// Store header for later processing
			hdrs[targetPath] = *hdr
			// Skip file data for now (will be processed in phase 2)
			if hdr.Typeflag == tar.TypeReg {
				if _, err := io.Copy(io.Discard, tr); err != nil {
					return fmt.Errorf("error skipping file %s: %v", targetPath, err)
				}
			}
		}
	}

	// Phase 2: Process non-directory entries in parallel
	eg := errgroup.Group{}
	eg.SetLimit(runtime.NumCPU() * 2) // Limit concurrent goroutines

	for targetPath, hdr := range hdrs {
		// Capture variables for goroutine
		targetPath, hdr := targetPath, hdr
		eg.Go(func() error {
			return processEntry(&hdr, targetPath, absDst)
		})
	}

	// Wait for all file processing to complete
	if err := eg.Wait(); err != nil {
		return err
	}

	// Phase 3: Set ownership and permissions for directories
	for _, dir := range dirs {
		if err := setOwnership(dir, hdrs); err != nil {
			logrus.Warnf("Error setting ownership for directory %s: %v", dir, err)
		}
	}

	return nil
}

// processEntry processes a single archive entry based on its type.
// It handles regular files, hard links, and symbolic links.
func processEntry(hdr *tar.Header, targetPath, rootfs string) error {
	switch hdr.Typeflag {
	case tar.TypeReg:
		// Regular file - need to re-read from original archive
		return processFiles(hdr, nil, targetPath)
	case tar.TypeLink:
		// Hard link
		return processLinks(hdr, rootfs, targetPath)
	case tar.TypeSymlink:
		// Symbolic link
		return processSymlinks(hdr, targetPath)
	default:
		logrus.Tracef("Skipping unsupported entry type %d for %s", hdr.Typeflag, targetPath)
		return nil
	}
}

// setOwnership sets the ownership of a file or directory if Chown is enabled.
// It extracts UID and GID from the tar header and applies them to the filesystem entry.
func setOwnership(path string, hdrs map[string]tar.Header) error {
	if !Chown {
		return nil // Skip ownership changes if not root
	}

	// Find the header for this path
	for headerPath, hdr := range hdrs {
		if headerPath == path {
			if err := os.Chown(path, hdr.Uid, hdr.Gid); err != nil {
				return fmt.Errorf("error setting ownership for %s: %v", path, err)
			}
			logrus.Tracef("Set ownership for %s to %d:%d", path, hdr.Uid, hdr.Gid)
			return nil
		}
	}

	// If no header found, log a warning but don't fail
	if err := os.Chown(path, 0, 0); err != nil {
		logrus.Warnf("Error setting owner for %s: %v", path, err)
	}
	return nil
}

// processFiles processes regular files from the archive.
// It checks if the file already exists with the same size and modification time
// to avoid unnecessary re-extraction.
func processFiles(hdr *tar.Header, src *tar.Reader, target string) error {
	// Check if file already exists with same size and modification time
	if info, err := os.Stat(target); err == nil {
		if info.Size() == hdr.Size && info.ModTime().Equal(hdr.ModTime) {
			logrus.Tracef("Skipping existing file: %s", target)
			// Skip the file data if src is provided
			if src != nil {
				if _, err := io.Copy(io.Discard, src); err != nil {
					return fmt.Errorf("error skipping file %s: %v", target, err)
				}
			}
			return nil
		}
	}

	// Create the file with appropriate permissions
	outFile, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode())
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", target, err)
	}
	defer outFile.Close()

	// Copy the file data if src is provided
	if src != nil {
		if _, err := io.Copy(outFile, src); err != nil {
			return fmt.Errorf("error writing file %s: %v", target, err)
		}
	}

	logrus.Tracef("Extracted file: %s", target)
	return nil
}

// processLinks processes hard links from the archive.
// It creates a hard link from the target to the link destination.
func processLinks(hdr *tar.Header, rootfs, target string) error {
	// Construct the absolute path for the link target
	linkTargetPath := filepath.Join(rootfs, hdr.Linkname)
	logrus.Tracef("Creating hard link: %s -> %s", target, linkTargetPath)

	// Remove any existing file at the target location
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("error removing existing file %s: %v", target, err)
	}

	// Create the hard link
	if err := os.Link(linkTargetPath, target); err != nil {
		return fmt.Errorf("error creating hard link %s: %v", target, err)
	}

	logrus.Tracef("Created hard link: %s -> %s", target, hdr.Linkname)
	return nil
}

// processSymlinks processes symbolic links from the archive.
// It creates a symbolic link pointing to the specified target.
func processSymlinks(hdr *tar.Header, target string) error {
	logrus.Tracef("Creating symbolic link: %s -> %s", target, hdr.Linkname)

	// Remove any existing file at the target location
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("error removing existing file %s: %v", target, err)
	}

	// Create the symbolic link
	if err := os.Symlink(hdr.Linkname, target); err != nil {
		return fmt.Errorf("error creating symbolic link %s: %v", target, err)
	}

	logrus.Tracef("Created symbolic link: %s -> %s", target, hdr.Linkname)
	return nil
}

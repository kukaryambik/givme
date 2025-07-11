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
// It processes the archive in sequential phases: first creating directories and extracting files,
// then processing other entry types like links in parallel for optimal performance.
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

	// Read entries and collect directories
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// End of archive
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

		// Create directories
		d := filepath.Dir(targetPath)
		if hdr.Typeflag == tar.TypeDir {
			d = targetPath
		}
		if !paths.PathContains(d, dirs) {
			// Check if the directory exists
			dstDirInfo, err := os.Stat(d)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error accessing %s: %v", d, err)
			}

			// Remove if it exists and is not a directory
			if dstDirInfo != nil && !dstDirInfo.IsDir() {
				if err := os.RemoveAll(d); err != nil {
					return fmt.Errorf("error removing %s: %v", d, err)
				}
			}

			// Create directory
			if err := os.MkdirAll(d, os.ModePerm); err != nil {
				return fmt.Errorf("error creating parent directory for %s: %v", targetPath, err)
			}
			dirs = append(dirs, d)
		}

		hdrs[targetPath] = *hdr

		if hdr.Typeflag == tar.TypeReg {
			if err := processFiles(hdr, tr, targetPath); err != nil {
				return err
			}
		}
	}

	// Process all non-directory entries in parallel
	processExceptDirs := func(name string, hdr tar.Header) error {
		switch hdr.Typeflag {
		case tar.TypeReg:
			restorePerm(name, &hdr)
			return nil
		case tar.TypeLink:
			return processLinks(&hdr, absDst, name)
		case tar.TypeSymlink:
			return processSymlinks(&hdr, name)
		default:
			return nil
		}
	}
	if err := parallelProcess(&hdrs, processExceptDirs); err != nil {
		return err
	}

	// Process directories to restore their permissions
	processDirs := func(name string, hdr tar.Header) error {
		if hdr.Typeflag == tar.TypeDir {
			restorePerm(name, &hdr)
		}
		return nil
	}
	if err := parallelProcess(&hdrs, processDirs); err != nil {
		return err
	}

	return nil
}

// parallelProcess processes tar archive entries in parallel using goroutines.
// It limits concurrency to the number of available CPU cores and executes
// the provided function for each entry in the headers map.
//
// Parameters:
//   - hdrs: pointer to map of file paths to tar headers
//   - fn: function to execute for each entry, receives path and header
//
// Returns:
//   - error: nil if all entries processed successfully, otherwise first error encountered
func parallelProcess(hdrs *map[string]tar.Header, fn func(string, tar.Header) error) error {
	sem := make(chan struct{}, runtime.NumCPU()) // Semaphore to limit concurrency
	var g errgroup.Group

	for name, hdr := range *hdrs {
		name := name
		hdr := hdr

		g.Go(
			func() error {
				sem <- struct{}{}        // Acquire a semaphore slot
				defer func() { <-sem }() // Release the semaphore slot
				return fn(name, hdr)
			},
		)
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

// restorePerm restores the permissions, timestamps, and ownership of a file or directory
// based on the information stored in the tar header. Ownership is only changed if
// the global Chown variable is true (typically when running as root).
//
// Parameters:
//   - path: filesystem path to the file or directory
//   - info: tar header containing the original file metadata
func restorePerm(path string, info *tar.Header) {
	// Restore file permissions
	if err := os.Chmod(path, info.FileInfo().Mode()); err != nil && !os.IsNotExist(err) {
		logrus.Warnf("Error setting permissions for %s: %v", path, err)
	}

	// Restore access and modification times
	if err := os.Chtimes(path, info.AccessTime, info.ModTime); err != nil && !os.IsNotExist(err) {
		logrus.Warnf("Error setting times for %s: %v", path, err)
	}

	// Restore ownership if running as root
	if Chown {
		if err := os.Chown(path, info.Uid, info.Gid); err != nil && !os.IsNotExist(err) {
			logrus.Warnf("Error setting owner for %s: %v", path, err)
		}
	}
}

// processFiles extracts regular files from the tar archive to the filesystem.
// It optimizes by skipping files that already exist with the same size and modification time.
// The function handles file creation, data copying, and basic error recovery.
//
// Parameters:
//   - hdr: tar header containing file metadata
//   - src: tar reader positioned at the file data
//   - target: destination filesystem path for the extracted file
//
// Returns:
//   - error: nil if file extracted successfully, otherwise describes the failure
func processFiles(hdr *tar.Header, src *tar.Reader, target string) error {

	// Check if file already exists with same properties to avoid unnecessary work
	if info, err := os.Stat(target); err == nil {
		// Check if the file already exists
		if info.Size() == hdr.Size && info.ModTime().Equal(hdr.ModTime) {
			logrus.Tracef("Skipping existing file: %s", target)
			if _, err := io.Copy(io.Discard, src); err != nil {
				return fmt.Errorf("error skipping file %s: %v", target, err)
			}
			return nil
		}
	}

	// Create the output file with appropriate permissions
	outFile, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode())
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", target, err)
	}

	// Copy the file data
	if _, err := io.Copy(outFile, src); err != nil {
		outFile.Close()
		return fmt.Errorf("error writing file %s: %v", target, err)
	}
	outFile.Close()

	logrus.Tracef("Extracted file: %s", target)

	return nil
}

// processLinks creates hard links from tar archive entries.
// A hard link creates multiple directory entries that point to the same inode,
// allowing the same file data to be accessed through different paths.
//
// Parameters:
//   - hdr: tar header containing link metadata and target path
//   - rootfs: root filesystem path for resolving relative link targets
//   - target: destination filesystem path where the hard link will be created
//
// Returns:
//   - error: nil if hard link created successfully, otherwise describes the failure
func processLinks(hdr *tar.Header, rootfs, target string) error {
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

	logrus.Tracef("Created link: %s -> %s", target, hdr.Linkname)

	return nil
}

// processSymlinks creates symbolic links from tar archive entries.
// A symbolic link is a special file that contains a path reference to another file or directory,
// allowing indirect access to the target through the link path.
//
// Parameters:
//   - hdr: tar header containing symlink metadata and target path
//   - target: destination filesystem path where the symbolic link will be created
//
// Returns:
//   - error: nil if symbolic link created successfully, otherwise describes the failure
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

	logrus.Tracef("Created link: %s -> %s", target, hdr.Linkname)

	return nil
}

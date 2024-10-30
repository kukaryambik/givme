package archiver

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

var Chown bool = os.Getuid() == 0

// Untar extracts a tar archive from `src` to `dst`, excluding any paths specified in `excl`.
func Untar(src io.Reader, dst string, excl []string) error {

	logrus.Debugf("Unpacking tar archive to %s", dst)

	absDst, err := filepath.Abs(dst)
	if err != nil {
		return fmt.Errorf("error getting absolute path for %s: %v", dst, err)
	}

	absExcl, err := paths.AbsAll(excl)
	if err != nil {
		return fmt.Errorf("failed to convert exclusion list to absolute paths: %v", err)
	}

	tr := tar.NewReader(src)

	hdrs := make(map[string]*tar.Header)
	dirs := make(map[string]bool)

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

		// Create parent directory
		parentDir := filepath.Dir(targetPath)
		if _, ok := dirs[parentDir]; !ok {
			if err := os.MkdirAll(parentDir, os.ModePerm); err != nil {
				return fmt.Errorf("error creating parent directory for %s: %v", targetPath, err)
			}
			dirs[parentDir] = true
		}

		switch hdr.Typeflag {
		case tar.TypeReg:
			if err := processFiles(hdr, tr, targetPath); err != nil {
				return err
			}
		case tar.TypeDir:
			if _, ok := dirs[targetPath]; !ok {
				if err := os.MkdirAll(targetPath, hdr.FileInfo().Mode()); err != nil {
					return fmt.Errorf("error creating directory: %v", err)
				}
				dirs[targetPath] = true
			}
			hdrs[targetPath] = hdr
		default:
			hdrs[targetPath] = hdr
		}
	}

	numCPU := runtime.NumCPU()
	sem := make(chan struct{}, numCPU)
	var g errgroup.Group

	for name, hdr := range hdrs {
		name := name
		hdr := hdr

		sem <- struct{}{} // Acquire a semaphore slot

		g.Go(func() error {
			defer func() { <-sem }() // Release the semaphore slot

			switch hdr.Typeflag {
			case tar.TypeDir:
				restorePerm(name, hdr)
				return nil
			case tar.TypeLink:
				return processLinks(hdr, absDst, name)
			case tar.TypeSymlink:
				return processSymlinks(hdr, name)
			case tar.TypeFifo:
				return processSpecialFiles(hdr, name)
			default:
				return nil
			}
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

// restorePerm restores the permissions of a file or directory
func restorePerm(path string, info *tar.Header) {
	if err := os.Chmod(path, info.FileInfo().Mode()); err != nil && !os.IsNotExist(err) {
		logrus.Warnf("Error setting permissions for %s: %v", path, err)
	}

	if err := os.Chtimes(path, info.AccessTime, info.ModTime); err != nil && !os.IsNotExist(err) {
		logrus.Warnf("Error setting times for %s: %v", path, err)
	}

	if Chown {
		if err := os.Chown(path, info.Uid, info.Gid); err != nil && !os.IsNotExist(err) {
			logrus.Warnf("Error setting owner for %s: %v", path, err)
		}
	}
}

// processFiles processes regular files and special files like FIFOs.
func processFiles(hdr *tar.Header, src *tar.Reader, target string) error {

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

	restorePerm(target, hdr)

	logrus.Tracef("Extracted file: %s", target)

	return nil
}

// processLinks processes hard links from the archive.
func processLinks(hdr *tar.Header, rootfs, target string) error {
	linkTargetPath := filepath.Join(rootfs, hdr.Linkname)
	logrus.Tracef("Creating hard link: %s -> %s", target, linkTargetPath)
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("error removing existing file %s: %v", target, err)
	}
	if err := os.Link(linkTargetPath, target); err != nil {
		return fmt.Errorf("error creating hard link %s: %v", target, err)
	}

	logrus.Tracef("Created link: %s -> %s", target, hdr.Linkname)

	return nil
}

// processSymlinks processes symbolic links from the archive.
func processSymlinks(hdr *tar.Header, target string) error {
	logrus.Tracef("Creating symbolic link: %s -> %s", target, hdr.Linkname)
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("error removing existing file %s: %v", target, err)
	}
	if err := os.Symlink(hdr.Linkname, target); err != nil {
		return fmt.Errorf("error creating symbolic link %s: %v", target, err)
	}

	logrus.Tracef("Created link: %s -> %s", target, hdr.Linkname)

	return nil
}

// processSpecialFiles handles special files like FIFOs during extraction.
func processSpecialFiles(hdr *tar.Header, target string) error {
	err := syscall.Mkfifo(target, uint32(hdr.FileInfo().Mode()))
	if err != nil {
		return fmt.Errorf("error creating FIFO %s: %v", target, err)
	}
	restorePerm(target, hdr)
	logrus.Tracef("Created FIFO: %s", target)

	return nil
}

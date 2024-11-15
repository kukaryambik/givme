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

	hdrs := make(map[string]tar.Header)
	var dirs []string

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

// parallelProcess processes files in parallel
func parallelProcess(hdrs *map[string]tar.Header, fn func(string, tar.Header) error) error {
	sem := make(chan struct{}, runtime.NumCPU())
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

// processFiles processes regular files from the archive
func processFiles(hdr *tar.Header, src *tar.Reader, target string) error {

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

// processLinks processes hard links from the archive
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

// processSymlinks processes symbolic links from the archive
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

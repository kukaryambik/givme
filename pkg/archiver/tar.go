package archiver

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

// fileIdentity uniquely identifies a file using device ID and inode number.
type fileIdentity struct {
	dev uint64
	ino uint64
}

// Tar creates a tar archive from the source directory `src` and saves it to `dst`,
// excluding any paths specified in `excl`.
func Tar(src, dst string, excl []string) error {
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for src %s: %v", src, err)
	}

	absDst, err := filepath.Abs(dst)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for dst %s: %v", dst, err)
	}

	absExcl, err := util.AbsAll(excl)
	if err != nil {
		return fmt.Errorf("failed to convert exclusion list to absolute paths: %v", err)
	}

	outFile, err := os.Create(absDst)
	if err != nil {
		logrus.Errorf("Error creating archive file %s: %v", absDst, err)
		return err
	}
	defer outFile.Close()

	tarWriter := tar.NewWriter(outFile)
	defer func() {
		if cerr := tarWriter.Close(); cerr != nil {
			logrus.Errorf("Error closing tar writer: %v", cerr)
		}
	}()

	addedFiles := make(map[fileIdentity]string)

	getFileID := func(fi os.FileInfo) (fileIdentity, error) {
		stat, ok := fi.Sys().(*syscall.Stat_t)
		if !ok {
			return fileIdentity{}, fmt.Errorf("unable to get raw syscall.Stat_t data for %s", fi.Name())
		}
		return fileIdentity{dev: uint64(stat.Dev), ino: uint64(stat.Ino)}, nil
	}

	err = filepath.Walk(absSrc, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			logrus.Errorf("Error accessing file %s: %v", file, err)
			return err
		}

		relPath, err := filepath.Rel(absSrc, file)
		if err != nil {
			logrus.Errorf("Error getting relative path for %s: %v", file, err)
			return err
		}

		if util.IsPathFrom(file, absExcl) {
			logrus.Tracef("Excluding: %s", file)
			if fi.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		hdr, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			logrus.Errorf("Error creating tar header for %s: %v", file, err)
			return err
		}
		hdr.Name = relPath

		if fi.Mode().IsRegular() {
			id, err := getFileID(fi)
			if err != nil {
				logrus.Warnf("Skipping file %s: %v", file, err)
				return nil
			}

			if fi.Sys() != nil {
				stat, _ := fi.Sys().(*syscall.Stat_t)
				if stat.Nlink > 1 {
					if original, exists := addedFiles[id]; exists {
						hdr.Typeflag = tar.TypeLink
						hdr.Linkname = original
						if err := tarWriter.WriteHeader(hdr); err != nil {
							logrus.Errorf("Error writing hard link header for %s: %v", file, err)
							return err
						}
						logrus.Tracef("Added hard link: %s -> %s", relPath, original)
						return nil
					}
					addedFiles[id] = relPath
				}
			}
		}

		if fi.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(file)
			if err != nil {
				logrus.Errorf("Error reading symlink %s: %v", file, err)
				return err
			}
			hdr.Linkname = linkTarget
		}

		if err := tarWriter.WriteHeader(hdr); err != nil {
			logrus.Errorf("Error writing header for %s: %v", file, err)
			return err
		}

		if fi.Mode().IsRegular() {
			f, err := os.Open(file)
			if err != nil {
				logrus.Errorf("Error opening file %s: %v", file, err)
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tarWriter, f); err != nil {
				logrus.Errorf("Error writing file %s to archive: %v", file, err)
				return err
			}
			logrus.Tracef("Added file: %s", relPath)
		}

		// Optionally, handle permissions, ownership, and timestamps here if needed.

		return nil
	})

	if err != nil {
		logrus.Errorf("Error walking source directory %s: %v", absSrc, err)
		return err
	}

	logrus.Debugf("Archive successfully created: %s", absDst)
	return nil
}

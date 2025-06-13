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

// fileIdentity uniquely identifies a file using device ID and inode number.
type fileIdentity struct {
	dev uint64
	ino uint64
}

// tarArchiver encapsulates the data and methods required for creating a tar archive.
type tarArchiver struct {
	absSrc     string
	absExcl    []string
	tarWriter  *tar.Writer
	addedFiles map[fileIdentity]string
}

// newTarArchiver initializes and returns a new tarArchiver instance.
func newTarArchiver(absSrc string, absExcl []string, tarWriter *tar.Writer) *tarArchiver {
	return &tarArchiver{
		absSrc:     absSrc,
		absExcl:    absExcl,
		tarWriter:  tarWriter,
		addedFiles: make(map[fileIdentity]string),
	}
}

// getFileID retrieves the file identity based on its FileInfo.
func (ta *tarArchiver) getFileID(fi os.FileInfo) (fileIdentity, error) {
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return fileIdentity{}, fmt.Errorf("unable to get raw syscall.Stat_t data for %s", fi.Name())
	}
	return fileIdentity{dev: uint64(stat.Dev), ino: uint64(stat.Ino)}, nil
}

// writeFileToTar writes a regular file to the tar archive.
func (ta *tarArchiver) writeFileToTar(file string, hdr *tar.Header) error {
	if err := ta.tarWriter.WriteHeader(hdr); err != nil {
		return fmt.Errorf("error writing header for %s: %v", file, err)
	}
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("error opening file %s: %v", file, err)
	}
	defer f.Close()

	if _, err := io.Copy(ta.tarWriter, f); err != nil {
		return fmt.Errorf("error writing file %s to archive: %v", file, err)
	}
	return nil
}

// handleRegularFile processes regular files, handling hard links if necessary.
func (ta *tarArchiver) handleRegularFile(file string, fi os.FileInfo, relPath string, hdr *tar.Header) error {
	id, err := ta.getFileID(fi)
	if err != nil {
		logrus.Warnf("Skipping file %s: %v", file, err)
		return nil
	}

	stat, ok := fi.Sys().(*syscall.Stat_t)
	if ok && stat.Nlink > 1 {
		if original, exists := ta.addedFiles[id]; exists {
			hdr.Typeflag = tar.TypeLink
			hdr.Linkname = original
			hdr.Size = 0

			if err := ta.tarWriter.WriteHeader(hdr); err != nil {
				return fmt.Errorf("error writing hard link header for %s: %v", file, err)
			}
			logrus.Tracef("Added hard link: %s -> %s", relPath, original)
			return nil
		}
		ta.addedFiles[id] = relPath
	}

	if err := ta.writeFileToTar(file, hdr); err != nil {
		return err
	}
	logrus.Tracef("Added file: %s", relPath)
	return nil
}

// handleSymlink processes symbolic links.
func (ta *tarArchiver) handleSymlink(file string, hdr *tar.Header) error {
	linkTarget, err := os.Readlink(file)
	if err != nil {
		return fmt.Errorf("error reading symlink %s: %v", file, err)
	}
	hdr.Linkname = linkTarget
	if err := ta.tarWriter.WriteHeader(hdr); err != nil {
		return fmt.Errorf("error writing header for symlink %s: %v", file, err)
	}
	return nil
}

// walkFunc is the function called by filepath.Walk.
func (ta *tarArchiver) walkFunc(file string, fi os.FileInfo, err error) error {
	if err != nil {
		logrus.Errorf("Error accessing file %s: %v", file, err)
		return err
	}

	relPath, err := filepath.Rel(ta.absSrc, file)
	if err != nil {
		logrus.Errorf("Error getting relative path for %s: %v", file, err)
		return err
	}

	if paths.PathFrom(file, ta.absExcl) {
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

	switch {
	case fi.Mode().IsRegular():
		if err := ta.handleRegularFile(file, fi, relPath, hdr); err != nil {
			logrus.Errorf("Error handling regular file %s: %v", file, err)
			return err
		}
	case fi.Mode()&os.ModeSymlink != 0:
		if err := ta.handleSymlink(file, hdr); err != nil {
			logrus.Errorf("Error handling symlink %s: %v", file, err)
			return err
		}
	case fi.Mode()&os.ModeNamedPipe != 0:
		hdr.Typeflag = tar.TypeFifo
		if err := ta.tarWriter.WriteHeader(hdr); err != nil {
			logrus.Errorf("Error writing header for FIFO %s: %v", file, err)
			return err
		}
		logrus.Tracef("Added FIFO: %s", relPath)
	default:
		if err := ta.tarWriter.WriteHeader(hdr); err != nil {
			logrus.Errorf("Error writing header for %s: %v", file, err)
			return err
		}
	}

	return nil
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

	absExcl, err := paths.AbsAll(excl)
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
		if err := tarWriter.Close(); err != nil {
			logrus.Errorf("Error closing tar writer: %v", err)
		}
	}()

	ta := newTarArchiver(absSrc, absExcl, tarWriter)

	err = filepath.Walk(absSrc, ta.walkFunc)
	if err != nil {
		logrus.Errorf("Error walking source directory %s: %v", absSrc, err)
		return err
	}

	logrus.Debugf("Archive successfully created: %s", absDst)
	return nil
}

package main

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func getMountedDirs() ([]string, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dirs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			dirs = append(dirs, fields[1])
		}
	}
	return dirs, scanner.Err()
}

func isExcluded(path string, excludedDirs []string) bool {
	if path == "/" {
		return false
	}
	for _, exclude := range excludedDirs {
		if path == exclude || strings.HasPrefix(path, exclude+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func isDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func addToArchive(path, source string, tarWriter *tar.Writer) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(source, path)
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(path)
		if err != nil {
			return err
		}
		header.Linkname = linkTarget
	}

	header.Name = relPath

	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		header.Uid, header.Gid, header.Mode = int(stat.Uid), int(stat.Gid), int64(stat.Mode)
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if fi.Mode().IsRegular() {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tarWriter, f)
		return err
	}

	return nil
}

func archiveAndDeleteRecursively(path, source string, tarWriter *tar.Writer, excludedDirs []string) error {
	if isExcluded(path, excludedDirs) {
		return nil
	}

	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if fi.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := archiveAndDeleteRecursively(filepath.Join(path, entry.Name()), source, tarWriter, excludedDirs); err != nil {
				return err
			}
		}

		isEmpty, err := isDirEmpty(path)
		if err != nil {
			return err
		}

		if err := addToArchive(path, source, tarWriter); err != nil {
			return fmt.Errorf("Error adding %s to archive: %v", path, err)
		}

		if isEmpty && path != "/" {
			if err := os.Remove(path); err != nil {
				fmt.Printf("Error removing directory %s: %v\n", path, err)
			}
		}
	} else {
		if err := addToArchive(path, source, tarWriter); err != nil {
			return fmt.Errorf("Error adding %s to archive: %v", path, err)
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			if err := os.Remove(path); err != nil {
				fmt.Printf("Error removing file %s: %v\n", path, err)
			}
		}
	}

	return nil
}

func main() {
	mountedDirs, err := getMountedDirs()
	if err != nil {
		fmt.Printf("Error retrieving mounted directories: %v\n", err)
		return
	}

	excludedDirs := append(
		mountedDirs,
		"/proc", "/sys", "/dev", "/run", // System directories
		"/busybox", "/workspace", "/rumett", // User-specific exclusions
	)

	source := "/"
	target := "/workspace/backup.tar"

	tarFile, err := os.Create(target)
	if err != nil {
		fmt.Printf("Error creating archive file: %v\n", err)
		return
	}
	defer tarFile.Close()

	tarWriter := tar.NewWriter(tarFile)
	defer tarWriter.Close()

	if err := archiveAndDeleteRecursively(source, source, tarWriter, excludedDirs); err != nil {
		fmt.Printf("Error during archiving and deletion: %v\n", err)
	} else {
		fmt.Println("Archiving and deletion completed successfully.")
	}
}

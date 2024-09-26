package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Rmrf(paths []string) error {
	for _, path := range paths {
		err := os.RemoveAll(path)
		if err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}
	return nil
}

func GetExecDir() string {
	exe, err := os.Executable()
	if err != nil {
		exe = "."
	}
	dir := filepath.Dir(exe)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return absDir
}

// GetMountDirs returns a list of mounted directories.
func GetMounts() ([]string, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dirs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] != "/" {
			dirs = append(dirs, fields[1])
		}
	}
	return dirs, scanner.Err()
}

// IsPathFrom checks if a path is from one of the listed paths.
func IsPathFrom(path string, list []string) bool {
	a, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		return false
	}
	for _, s := range list {
		b, err := filepath.Abs(s)
		if err != nil {
			fmt.Printf("Error getting absolute path: %v\n", err)
			return false
		}
		if a == b || strings.HasPrefix(a, b+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// IsPathContains checks if a path contains one of the listed paths.
func IsPathContains(path string, list []string) bool {
	a, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		return false
	}
	if path == "/" {
		return true
	}
	for _, s := range list {
		b, err := filepath.Abs(s)
		if err != nil {
			return false
		}
		if a == b || strings.HasPrefix(b, a+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// IsDirEmpty checks if a directory is empty.
func IsDirEmpty(dir string) (bool, error) {
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

package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Slugify converts a string into a slug (URL-friendly format).
func Slugify(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return strings.Trim(re.ReplaceAllString(s, "-"), "-")
}

// Rmrf removes all files and directories in the provided paths.
func Rmrf(paths []string) error {
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}
	return nil
}

// GetExecDir returns the directory of the current executable.
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

// GetMounts returns a list of mounted directories.
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

// IsPathFrom checks if a path originates from any of the listed paths.
func IsPathFrom(path string, list []string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		return false
	}
	for _, base := range list {
		absBase, err := filepath.Abs(base)
		if err != nil {
			fmt.Printf("Error getting absolute path: %v\n", err)
			return false
		}
		if absPath == absBase || strings.HasPrefix(absPath, absBase+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// IsPathContains checks if a path contains any of the listed paths.
func IsPathContains(path string, list []string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		return false
	}
	if path == "/" {
		return true
	}
	for _, subPath := range list {
		absSubPath, err := filepath.Abs(subPath)
		if err != nil {
			return false
		}
		if absPath == absSubPath || strings.HasPrefix(absSubPath, absPath+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// IsDirEmpty checks if the specified directory is empty.
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

package paths

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// Rmrf removes all files and directories in the provided paths.
var Rmrf = func(paths ...string) error {
	for _, path := range paths {
		err := os.RemoveAll(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}
	return nil
}

// GetExecDir returns the directory of the current executable.
var GetExecDir = func() string {
	exe, err := os.Executable()
	if err != nil {
		exe = "."
	}
	dir := filepath.Dir(exe)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		logrus.Warnf("failed to get absolute path: %v", err)
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

func AbsAll(paths []string) ([]string, error) {
	var absPaths []string

	for _, p := range paths {
		ap, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}
		absPaths = append(absPaths, ap)
	}

	return absPaths, nil
}

// IsPathFrom checks if a path originates from any of the listed paths.
var IsPathFrom = func(path string, list []string) bool {
	for _, base := range list {
		if path == base || strings.HasPrefix(path, base+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// IsPathContains checks if a path contains any of the listed paths.
var IsPathContains = func(path string, list []string) bool {
	for _, c := range list {
		p := strings.TrimRight(path, string(os.PathSeparator)) + string(os.PathSeparator)
		if path == c || strings.HasPrefix(c, p) {
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

// IsFileExists checks if the specified file exists.
var IsFileExists = func(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

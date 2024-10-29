package paths

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// Rmrf removes all files and directories in the provided paths.
var Rmrf = func(path string, ignore []string) error {

	// List all paths
	var lst []string
	if err := GetList(path, ignore, &lst); err != nil {
		return err
	}

	logrus.Debugf("Removing paths: %v", lst)
	for _, path := range lst {
		err := os.RemoveAll(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}
	return nil
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

// PathFrom checks if a path originates from any of the listed paths.
var PathFrom = func(path string, list []string) bool {
	for _, base := range list {
		if path == base || strings.HasPrefix(path, base+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// PathContains checks if a path contains any of the listed paths.
var PathContains = func(path string, list []string) bool {
	for _, c := range list {
		p := strings.TrimRight(path, string(os.PathSeparator)) + string(os.PathSeparator)
		if path == c || strings.HasPrefix(c, p) {
			return true
		}
	}
	return false
}

// FileExists checks if the specified file exists.
var FileExists = func(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

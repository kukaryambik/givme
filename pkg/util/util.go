package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Retry attempts to execute a function multiple times with delay between attempts
func Retry(attempts int, sleep time.Duration, fn func() error) error {
	var err error
	for i := 1; i <= attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}

		logrus.Warnf("attempt %d/%d failed: %v", i, attempts, err)

		if i < attempts {
			time.Sleep(sleep * time.Duration(i))
		}
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func MergeStructs(src, dst interface{}, overwrite ...bool) {
	logrus.Debugf("Merging structs: %v, %v", &src, &dst)
	srcVal := reflect.ValueOf(src).Elem() // Get Value for reading fields
	dstVal := reflect.ValueOf(dst).Elem() // Get Value for setting fields

	for i := 0; i < dstVal.NumField(); i++ {
		srcField := srcVal.Field(i)
		dstField := dstVal.Field(i)

		// Check if the source field is not zero (non-empty)
		if !srcField.IsZero() && dstField.CanSet() {
			if len(overwrite) > 0 && overwrite[0] {
				dstField.Set(srcField)
			} else if dstField.IsZero() {
				dstField.Set(srcField)
			}
		}
	}
	logrus.Debugf("Merged struct: %v", &dst)
}

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
func IsPathFrom(path string, list []string) (bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("error getting absolute path for %s: %v", path, err)
	}
	for _, base := range list {
		absBase, err := filepath.Abs(base)
		if err != nil {
			return false, fmt.Errorf("error getting absolute path for %s: %v", base, err)
		}
		if absPath == absBase || strings.HasPrefix(absPath, absBase+string(os.PathSeparator)) {
			return true, nil
		}
	}
	return false, nil
}

// IsPathContains checks if a path contains any of the listed paths.
func IsPathContains(path string, list []string) (bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("error getting absolute path for %s: %v", path, err)
	}
	if path == "/" {
		return true, nil
	}
	for _, subPath := range list {
		absSubPath, err := filepath.Abs(subPath)
		if err != nil {
			return false, fmt.Errorf("error getting absolute path for %s: %v", subPath, err)
		}
		if absPath == absSubPath || strings.HasPrefix(absSubPath, absPath+string(os.PathSeparator)) {
			return true, nil
		}
	}
	return false, nil
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
func IsFileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

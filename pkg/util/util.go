package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// UniqString creates an array of string with unique values.
func UniqString(a []string) []string {
	var (
		length  = len(a)
		seen    = make(map[string]struct{}, length)
		results = make([]string, 0)
	)

	for i := 0; i < length; i++ {
		v := a[i]

		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		results = append(results, v)
	}

	return results
}

// Retry attempts to execute a function multiple times with delay between attempts
func Retry(retries int, sleep time.Duration, fn func() error) error {
	var err error
	for i := 0; i <= retries; i++ {
		if err = fn(); err == nil {
			return nil
		}

		logrus.Warnf("attempt %d/%d failed: %v", i, retries, err)

		if i < retries {
			time.Sleep(sleep * time.Duration(i))
		}
	}
	return fmt.Errorf("after %d attempts, last error: %s", retries, err)
}

// Slugify converts a string into a slug (URL-friendly format).
func Slugify(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return strings.Trim(re.ReplaceAllString(s, "-"), "-")
}

// Rmrf removes all files and directories in the provided paths.
func Rmrf(paths ...string) error {
	for _, path := range paths {
		err := os.RemoveAll(path)
		if err != nil && !os.IsNotExist(err) {
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
func IsPathFrom(path string, list []string) bool {
	for _, base := range list {
		if path == base || strings.HasPrefix(path, base+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// IsPathContains checks if a path contains any of the listed paths.
func IsPathContains(rootpath, path string, list []string) bool {
	if path == rootpath {
		return true
	}
	for _, subPath := range list {
		if path == subPath || strings.HasPrefix(subPath, path+string(os.PathSeparator)) {
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
func IsFileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

package util

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// GetParentProcessName returns the name of the parent process or an empty string
// if the parent does not exist.
func GetParentProcessName() (string, error) {
	ppid := os.Getppid()
	commPath := filepath.Join("/proc", strconv.Itoa(ppid), "comm")
	comm, err := os.ReadFile(commPath)
	if err != nil {
		if os.IsNotExist(err) {
			// The parent process does not exist, return an empty string.
			return "", nil
		}

		return "", fmt.Errorf("failed to read %s: %w", commPath, err)
	}

	// Trim the process name to remove any trailing newlines.
	return strings.TrimSpace(string(comm)), nil
}

// IsOutRedirected returns true if the standard output is not a terminal.
// This is done by checking if the standard output is a character device.
func IsOutRedirected() bool {
	// Get the file info of the standard output.
	stdoutInfo, err := os.Stdout.Stat()
	if err != nil {
		// If an error occurred, assume it is a terminal.
		return false
	}

	// Check if the standard output is a character device.
	// If it is, it means it is a terminal.
	return (stdoutInfo.Mode() & os.ModeCharDevice) == 0
}

// GetExecDir returns the directory of the current executable.
var GetExecDir = sync.OnceValue(func() string {
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
})

// Coalesce returns the first non-zero value from the given values.
// If all values are zero, it returns the zero value of the type.
func Coalesce[T any](vals ...T) T {
	// Initialize the result with the zero value of the type.
	var zeroValue T

	// Iterate over the given values.
	for _, v := range vals {
		// Check if the value is not zero.
		if !reflect.ValueOf(v).IsZero() {
			// Return the first non-zero value.
			return v
		}
	}

	// Return the zero value if all values are zero.
	return zeroValue
}

// CleanList removes all zero values from the given list.
//
// It iterates over the list and checks if each value is not zero using the reflect package.
// If the value is not zero, it is added to the result list.
// The function is useful for cleaning up lists of values returned by APIs or functions
// that might return zero values.
func CleanList[T any](list []T) []T {
	var result []T
	for _, v := range list {
		r := reflect.ValueOf(v)
		if !r.IsZero() {
			result = append(result, v)
		}
	}
	return result
}

// UniqString creates an array of string with unique values.
// It iterates over the given list and creates a map of strings as keys.
// If the string is not present in the map, it is added to the map
// and the result list.
func UniqString(strs []string) []string {
	seen := make(map[string]struct{}, len(strs))
	var uniqueStrs []string

	// Iterate over the given list.
	for _, s := range strs {
		// Check if the string is not present in the map.
		if _, ok := seen[s]; !ok {
			// Add the string to the map.
			seen[s] = struct{}{}
			// Add the string to the result list.
			uniqueStrs = append(uniqueStrs, s)
		}
	}

	// Return the result list.
	return uniqueStrs
}

// Slugify converts a string into a slug (URL-friendly format).
//
// The function uses a regular expression to replace all non-alphanumeric characters
// with a hyphen and then trims the result to remove any leading or trailing hyphens.
func Slugify(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return strings.Trim(re.ReplaceAllString(s, "-"), "-")
}

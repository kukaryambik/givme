package util

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

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

func Coalesce[T any](vals ...T) T {
	var zero T
	for _, v := range vals {
		r := reflect.ValueOf(v)
		if !r.IsZero() {
			return v
		}
	}
	return zero
}

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

// Slugify converts a string into a slug (URL-friendly format).
func Slugify(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return strings.Trim(re.ReplaceAllString(s, "-"), "-")
}

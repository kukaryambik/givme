package util

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

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

func PrepareEnv(env []string) []string {
	// Add the current exec directory to PATH
	path := slices.IndexFunc(env, func(s string) bool {
		return strings.HasPrefix(s, "PATH=")
	})
	logrus.Debugf("PATH index: %d", path)
	if path != -1 {
		env[path] = env[path] + ":" + GetExecDir()
	}

	// Format environment variables for export
	for n, e := range env {
		kv := strings.SplitN(e, "=", 2)
		env[n] = fmt.Sprintf("export %s=%s", kv[0], strconv.Quote(kv[1]))
	}

	logrus.Debugf("Environments variables: %s", env)
	return env
}

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

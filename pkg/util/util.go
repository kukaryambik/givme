package util

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

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

// Retry attempts to execute a function multiple times with delay between attempts
var Retry = func(retries int, sleep time.Duration, fn func() error) error {
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

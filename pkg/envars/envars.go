package envars

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

func FromFile(new map[string]string, file string, overwrite bool) (map[string]string, error) {
	// Reading variables from file
	old, err := godotenv.Read(file)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error reading file %s: %v", file, err)
	}

	if overwrite {
		if err := godotenv.Write(new, file); err != nil {
			return nil, fmt.Errorf("error writing to file %s: %v", file, err)
		}
	}

	return old, nil
}

// ToMap parsing environment variables into a map
func ToMap(env []string) map[string]string {
	envMap := make(map[string]string, len(env))
	for _, e := range env {
		kv := strings.SplitN(e, "=", 2)
		if len(kv) == 2 {
			envMap[kv[0]] = kv[1]
		}
	}
	return envMap
}

// ToSlice converting map into a slice of strings
func ToSlice(quote bool, m map[string]string) []string {
	slice := make([]string, 0, len(m))
	for k, v := range m {
		if quote {
			v = strconv.Quote(v)
		}
		slice = append(slice, k+"="+v)
	}
	return slice
}

// Uniq returns environment variables unique for x or duplicates
func Uniq(duplicates bool, x, y map[string]string) map[string]string {
	z := make(map[string]string, len(x))
	for xKey, xVal := range x {
		if yVal, yKeyExists := y[xKey]; yKeyExists == duplicates && (!yKeyExists || xVal == yVal) {
			z[xKey] = xVal
		}
	}
	return z
}

// UniqKeys returns environment variables from x that are not present in y
func UniqKeys(x, y map[string]string) map[string]string {
	z := make(map[string]string, len(x))
	for k, v := range x {
		if _, exists := y[k]; !exists {
			z[k] = v
		}
	}
	return z
}

// Merge merges maps
func Merge(maps ...map[string]string) map[string]string {
	z := make(map[string]string, len(maps)*(len(maps[0])/2))
	for _, m := range maps {
		for k, v := range m {
			z[k] = v
		}
	}
	return z
}

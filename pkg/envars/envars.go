package envars

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// PrepareEnv prepares the environment variables for the container
func PrepareToEval(unset, set map[string]string) string {
	var unsetStr, setStr string
	if len(unset) > 0 {
		unsetStr = "unset " + strings.Join(ToSlice(true, unset), " ") + ";"
	}
	if len(set) > 0 {
		setStr = "export " + strings.Join(ToSlice(true, set), " ") + ";"
	}
	return strings.TrimSpace(fmt.Sprintf("%s\n%s", unsetStr, setStr))
}

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
	envMap := make(map[string]string)
	for _, e := range env {
		if kv := strings.SplitN(e, "=", 2); len(kv) == 2 {
			envMap[kv[0]] = kv[1]
		}
	}
	return envMap
}

// ToSlice converting map into a slice of strings
func ToSlice(quote bool, m map[string]string) []string {
	var slice []string
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
	z := make(map[string]string)
	for xKey, xVal := range x {
		if yVal, yKeyExists := y[xKey]; (yKeyExists && xVal == yVal) == duplicates {
			z[xKey] = xVal
		}
	}
	return z
}

// UniqKeys returns environment variables from x that are not present in y
func UniqKeys(x, y map[string]string) map[string]string {
	z := make(map[string]string)
	for xKey := range x {
		if _, exists := y[xKey]; !exists {
			z[xKey] = x[xKey]
		}
	}
	return z
}

// Merge merges maps
func Merge(maps ...map[string]string) map[string]string {
	z := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			z[k] = v
		}
	}
	return z
}

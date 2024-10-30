package envars

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

var defaultKeepEnv = []string{
	"PWD",
	"SHLVL",
	"USER",
	"TERM",
	"SSL_CERT_DIR",
}

// AddToPath adds a path to the end of PATH environment variable
func AddToPath(env []string, path string) []string {
	p := slices.IndexFunc(env, func(s string) bool {
		return strings.HasPrefix(s, "PATH=")
	})
	if p > -1 {
		env[p] = env[p] + ":" + path
	} else {
		env = append(env, "PATH="+path)
	}
	return env
}

// PrepareEnv prepares environment variables for applying to the container
func PrepareEnv(file string, overwrite bool, env []string) []string {
	var finalEnv []string

	// Read variables from the previous image from the file
	fileMap, err := godotenv.Read(file)
	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
		logrus.Warnf("Error reading file %s: %v", file, err)
	}

	// Create a map of new variables
	newMap := Split(env)
	logrus.Debugf("New environment variables: %s", newMap)

	// Save new variables to the file if overwrite is enabled
	if overwrite {
		if err := godotenv.Write(newMap, file); err != nil {
			logrus.Warnf("Error writing to file %s: %v", file, err)
		}
	}

	// Get current environment variables
	currentMap := Split(os.Environ())
	logrus.Debugf("Environment variables: %s", currentMap)

	// Determine which variables were set from the previous image
	duplicatesMap := GetDuplicates(currentMap, fileMap)
	logrus.Debugf("Duplicate environment variables: %s", duplicatesMap)

	// They need to be cleared
	for k := range duplicatesMap {
		if !slices.Contains(defaultKeepEnv, k) {
			finalEnv = append(finalEnv, fmt.Sprintf("unset %s", k))
		}
	}

	// Get the list of variables set after the container started
	currentWithoutDuplicates := UniqKeys(currentMap, duplicatesMap)
	logrus.Debugf("Environment variables without duplicates: %s", currentWithoutDuplicates)

	// Determine the list of new unique variables
	uniqMap := UniqKeys(newMap, currentWithoutDuplicates)
	logrus.Debugf("UniqMap variables: %s", uniqMap)

	// Add PATH
	if _, exists := newMap["PATH"]; exists {
		uniqMap["PATH"] = newMap["PATH"] + ":" + util.GetExecDir()
		logrus.Debugf("PATH is %s", uniqMap["PATH"])
	}

	// Compile the list of new variables
	for k, v := range uniqMap {
		finalEnv = append(finalEnv, fmt.Sprintf("export %s=%s", k, strconv.Quote(v)))
	}

	logrus.Debugf("Final environment variables: %s", finalEnv)
	return finalEnv
}

// Split separates the environment variables into a map of key-value pairs
func Split(env []string) map[string]string {
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		envMap[key] = value
	}
	return envMap
}

// GetDuplicates returns environment variables present in both x and y
func GetDuplicates(x, y map[string]string) map[string]string {
	z := make(map[string]string)
	// Iterate over x to find duplicate keys
	for xKey, xVal := range x {
		if yVal, yKeyExists := y[xKey]; yKeyExists && xVal == yVal {
			z[xKey] = xVal
		}
	}
	return z
}

// UniqKeys returns environment variables from x that are not present in y
func UniqKeys(x, y map[string]string) map[string]string {
	z := make(map[string]string)
	// Iterate over x to find unique keys
	for xKey := range x {
		if _, exists := y[xKey]; !exists {
			z[xKey] = x[xKey]
		}
	}
	return z
}

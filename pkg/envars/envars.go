package envars

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func Getenv(s []string, key string) string {
	p := slices.IndexFunc(s, func(s string) bool {
		return strings.HasPrefix(s, key+"=")
	})
	if p > -1 {
		return strings.TrimPrefix(s[p], key+"=")
	}
	return ""
}

func Which(env []string, cmd string) (string, error) {
	oldPath := os.Getenv("PATH")
	p := Getenv(env, "PATH")
	if p == "" {
		return "", fmt.Errorf("failed to find PATH in env: %v", env)
	}
	os.Setenv("PATH", p)
	b, err := exec.LookPath(cmd)
	os.Setenv("PATH", oldPath)
	if err != nil {
		return "", fmt.Errorf("failed to find %s in PATH: %v", cmd, err)
	}
	return b, nil
}

func CoalesceWhich(env []string, cmd ...string) (string, error) {
	var paths []string
	for _, c := range cmd {
		p, _ := Which(env, c)
		paths = append(paths, p)
	}
	return util.Coalesce(paths...), nil
}

// PrepareEnv prepares environment variables for applying to the container
func PrepareEnv(file string, save, overwrite bool, env []string) ([]string, error) {
	// Read variables from the previous image from the file
	fileMap, err := godotenv.Read(file)
	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
		return nil, fmt.Errorf("error reading file %s: %v", file, err)
	}

	// Create a map of new variables
	newMap := Split(env)
	logrus.Debugf("New environment variables: %s", newMap)

	// Save new variables to the file
	if save {
		if err := godotenv.Write(newMap, file); err != nil {
			return nil, fmt.Errorf("error writing to file %s: %v", file, err)
		}
	}

	// Get current environment variables
	currentMap := Split(os.Environ())
	logrus.Debugf("Environment variables: %s", currentMap)

	// Determine which variables were set from the previous image
	uniqOldMap := Uniq(currentMap, fileMap)
	logrus.Debugf("Uniq environment variables: %s", uniqOldMap)

	// Determine the list of new unique variables
	var finalMap map[string]string
	if overwrite {
		finalMap = Merge(uniqOldMap, newMap)
	} else {
		finalMap = Merge(newMap, uniqOldMap)
	}
	logrus.Debugf("Uniq new variables: %s", finalMap)

	// Update PATH
	paths := util.CleanList([]string{newMap["PATH"], util.GetExecDir()})
	finalMap["PATH"] = strings.Join(paths, ":")

	// Compile the list of new variables
	var finalEnv []string
	for k, v := range finalMap {
		finalEnv = append(finalEnv, fmt.Sprintf("%s=%s", k, v))
	}

	logrus.Debugf("Final environment variables: %s", finalEnv)
	return finalEnv, nil
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

// Uniq returns uniq environment variables from x
func Uniq(x, y map[string]string) map[string]string {
	z := make(map[string]string)
	// Iterate over x to find duplicate keys
	for xKey, xVal := range x {
		if yVal, yKeyExists := y[xKey]; !yKeyExists || xVal != yVal {
			z[xKey] = xVal
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

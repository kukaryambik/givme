package env

import (
	"fmt"
	"os"
	"strings"
)

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

// DiffX returns environment variables present in both x and y
// using values from x.
func DiffX(x, y []string) []string {
	xMap := Split(x)
	yMap := Split(y)
	var z []string

	// Iterate over xMap to find matching keys in yMap
	for key, value := range xMap {
		if _, exists := yMap[key]; exists {
			// If key exists in y, add it to the result slice
			z = append(z, key+"="+value)
		}
	}
	return z
}

// SaveToFile writes the environment variables to the specified file
func SaveToFile(env []string, fileName string) error {
	// Create the file for writing
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", fileName, err)
	}
	defer file.Close()

	content := strings.Join(env, "\n") + "\n"

	// Write the content to the file
	_, err = file.WriteString(content)
	if err != nil {
		file.Close()
		os.Remove(fileName)
		return fmt.Errorf("error writing to file %s: %v", fileName, err)
	}

	return nil
}

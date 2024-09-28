package list

import (
	"os"
	"path/filepath"

	"github.com/kukaryambik/givme/pkg/util"
)

// ListPaths recursively lists files and directories, excluding specified paths.
func ListPaths(path string, exclude []string, list *[]string) error {
	absPath, err := filepath.Abs(path)
	if err != nil || util.IsPathFrom(absPath, exclude) {
		return err
	}

	// Get file or directory info
	fi, err := os.Lstat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// If it's a directory, process its contents recursively
	if fi.IsDir() {
		if util.IsPathContains(absPath, exclude) {
			entries, err := os.ReadDir(absPath)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				// Recursively list paths within the directory
				if err := ListPaths(filepath.Join(absPath, entry.Name()), exclude, list); err != nil {
					return err
				}
			}
		} else {
			*list = append(*list, absPath)
		}
	} else {
		// Add the file path to the list
		*list = append(*list, absPath)
	}

	return nil
}

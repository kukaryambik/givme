package listpaths

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

// List recursively lists files and directories, excluding specified paths.
func List(rootpath, path string, exclude []string, lst *[]string) error {
	absRoot, err := filepath.Abs(rootpath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", rootpath, err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}

	absExclude, err := util.AbsAll(exclude)
	if err != nil {
		return err
	}

	logrus.Debugf("Processing path: %s", absPath)

	// Get file or directory info
	fi, err := os.Lstat(absPath)
	if os.IsNotExist(err) {
		logrus.Debugf("Path %s does not exist", absPath)
		return nil
	} else if err != nil {
		return fmt.Errorf("error getting file info for path %s: %v", absPath, err)
	}

	// Check if the path should be excluded using util.IsPathFrom
	if util.IsPathFrom(absPath, absExclude) {
		logrus.Debugf("Path %s is excluded by IsPathFrom", absPath)
		return nil
	}

	// Check if the path contain some excludes in it
	if fi.IsDir() && util.IsPathContains(absRoot, absPath, absExclude) {
		logrus.Tracef("Path %s is a directory", absPath)

		// Read the contents of the directory
		entries, err := os.ReadDir(absPath)
		if err != nil {
			logrus.Errorf("Error reading directory %s: %v", absPath, err)
			return err
		}

		// Recursively process the contents of the directory
		for _, entry := range entries {
			logrus.Tracef("Recursively processing entry %s in directory %s",
				entry.Name(), absPath)
			if err := List(
				absRoot, filepath.Join(absPath, entry.Name()),
				absExclude, lst,
			); err != nil {
				return err
			}
		}
	} else {
		// Add the file path to the list
		logrus.Debugf("Adding path %s to the list", absPath)
		*lst = append(*lst, absPath)
	}

	return nil
}

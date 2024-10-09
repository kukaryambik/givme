package listpaths

import (
	"os"
	"path/filepath"

	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

// List recursively lists files and directories, excluding specified paths.
func List(path string, exclude []string, lst *[]string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		logrus.Errorf("Failed to get absolute path for %s: %v", path, err)
		return err
	}
	logrus.Debugf("Processing path: %s", absPath)

	// Get file or directory info
	fi, err := os.Lstat(absPath)
	if os.IsNotExist(err) {
		logrus.Debugf("Path %s does not exist", absPath)
		return nil
	} else if err != nil {
		logrus.Errorf("Error getting file info for path %s: %v", absPath, err)
		return err
	}

	// Check if the path should be excluded using util.IsPathFrom
	itsExcluded, err := util.IsPathFrom(absPath, exclude)
	if err != nil {
		logrus.Errorf("Error checking exclusion with IsPathFrom for path %s: %v",
			absPath, err)
		return err
	}
	if itsExcluded {
		logrus.Debugf("Path %s is excluded by IsPathFrom", absPath)
		return nil
	}

	// Check if the path contain some excludes in it
	pathHasExcludes, err := util.IsPathContains(absPath, exclude)
	if err != nil {
		logrus.Errorf("Error checking exclusion with IsPathContains for "+
			"directory %s: %v", absPath, err)
		return err
	}
	if fi.IsDir() && pathHasExcludes {
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
			if err := List(filepath.Join(
				absPath, entry.Name()), exclude, lst,
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

package list

import (
	"os"
	"path/filepath"

	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

// ListPaths recursively lists files and directories, excluding specified paths.
func ListPaths(
	logger *logrus.Logger, path string, exclude []string, list *[]string,
) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		logger.Errorf("Failed to get absolute path for %s: %v", path, err)
		return err
	}
	logger.Debugf("Processing path: %s", absPath)

	// Check if the path should be excluded using util.IsPathFrom
	shouldExclude, err := util.IsPathFrom(absPath, exclude)
	if err != nil {
		logger.Errorf("Error checking exclusion with IsPathFrom for path %s: %v",
			absPath, err)
		return err
	}
	if shouldExclude {
		logger.Debugf("Path %s is excluded by IsPathFrom", absPath)
		return nil
	}

	// Get file or directory info
	fi, err := os.Lstat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warnf("Path %s does not exist", absPath)
			return nil
		}
		logger.Errorf("Error getting file info for path %s: %v", absPath, err)
		return err
	}

	// If it's a directory, process its contents recursively
	if fi.IsDir() {
		logger.Tracef("Path %s is a directory", absPath)

		pathHasExcludes, err := util.IsPathContains(absPath, exclude)
		if err != nil {
			logger.Errorf("Error checking exclusion with IsPathContains for "+
				"directory %s: %v", absPath, err)
			return err
		}

		if pathHasExcludes {
			entries, err := os.ReadDir(absPath)
			if err != nil {
				logger.Errorf("Error reading directory %s: %v", absPath, err)
				return err
			}

			for _, entry := range entries {
				// Recursively list paths within the directory
				logger.Tracef("Recursively processing entry %s in directory %s",
					entry.Name(), absPath)
				if err := ListPaths(logger, filepath.Join(
					absPath, entry.Name()), exclude, list,
				); err != nil {
					return err
				}
			}
		} else {
			// Add the file path to the list
			logger.Debugf("Adding file path %s to the list", absPath)
			*list = append(*list, absPath)
		}
	} else {
		// Add the file path to the list
		logger.Debugf("Adding file path %s to the list", absPath)
		*list = append(*list, absPath)
	}

	return nil
}

// package list

// import (
// 	"os"
// 	"path/filepath"

// 	"github.com/kukaryambik/givme/pkg/util"
// )

// // ListPaths recursively lists files and directories, excluding specified paths.
// func ListPaths(path string, exclude []string, list *[]string) error {
// 	absPath, err := filepath.Abs(path)
// 	if err != nil || util.IsPathFrom(absPath, exclude) {
// 		return err
// 	}

// 	// Get file or directory info
// 	fi, err := os.Lstat(absPath)
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			return nil
// 		}
// 		return err
// 	}

// 	// If it's a directory, process its contents recursively
// 	if fi.IsDir() {
// 		if util.IsPathContains(absPath, exclude) {
// 			entries, err := os.ReadDir(absPath)
// 			if err != nil {
// 				return err
// 			}
// 			for _, entry := range entries {
// 				// Recursively list paths within the directory
// 				if err := ListPaths(filepath.Join(absPath, entry.Name()), exclude, list); err != nil {
// 					return err
// 				}
// 			}
// 		} else {
// 			*list = append(*list, absPath)
// 		}
// 	} else {
// 		// Add the file path to the list
// 		*list = append(*list, absPath)
// 	}

// 	return nil
// }

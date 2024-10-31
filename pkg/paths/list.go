package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// GetList recursively lists files and directories, excluding specified paths.
func GetList(path string, ignore []string, lst *[]string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}

	absExclude, err := AbsAll(ignore)
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

	// Check if the path should be ignored using util.PathFrom
	if PathFrom(absPath, absExclude) {
		logrus.Tracef("Path %s is ignored by PathFrom", absPath)
		return nil
	}

	// Check if the path contain some ignores in it
	if fi.IsDir() && PathContains(absPath, absExclude) {
		logrus.Tracef("Path %s is a directory", absPath)

		// Read the contents of the directory
		entries, err := os.ReadDir(absPath)
		if err != nil {
			logrus.Errorf("Error reading directory %s: %v", absPath, err)
			return err
		}

		// Recursively process the contents of the directory
		numCPU := runtime.NumCPU()
		sem := make(chan struct{}, numCPU)
		var g errgroup.Group

		for _, entry := range entries {
			entry := entry

			sem <- struct{}{} // Acquire a semaphore slot

			g.Go(func() error {
				defer func() { <-sem }() // Release the semaphore slot

				logrus.Tracef("Recursively processing entry %s in directory %s",
					entry.Name(), absPath)
				if err := GetList(
					filepath.Join(absPath, entry.Name()),
					absExclude, lst,
				); err != nil {
					return err
				}
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}

	} else {
		// Add the file path to the list
		logrus.Tracef("Adding path %s to the list", absPath)
		*lst = append(*lst, absPath)
	}

	return nil
}

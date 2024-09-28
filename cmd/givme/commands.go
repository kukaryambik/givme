package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/env"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/list"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

// buildExclusions generates a list of directories that should be excluded
// from operations such as snapshot creation or restoration.
func buildExclusions(logger *logrus.Logger) ([]string, error) {
	logger.Debugf("Building list of exclusions...")
	mounts, err := util.GetMounts() // Get system mount points.
	if err != nil {
		logger.Errorf("Error getting system mounts: %v", err)
		return nil, err
	}

	// Append system directories and user-defined paths to exclusions.
	exclusions := append(mounts, "/proc", "/sys", "/dev", "/run",
		"/busybox", workDir, file)

	logger.Tracef("Exclusions before adding user-defined: %v", exclusions)

	// Add user-specified excluded directories from environment variable.
	if exclude != "" {
		exclusions = append(exclusions, strings.FieldsFunc(
			exclude, func(r rune) bool {
				return r == ':' || r == ','
			})...)
		logger.Tracef("Exclusions after adding user-defined: %v", exclusions)
	}

	return exclusions, nil
}

// snapshotCmd creates a tar archive of the rootfs directory, excluding
// the directories specified in buildExclusions.
func snapshotCmd(logger *logrus.Logger) error {
	logger.Debugf("Starting snapshot creation...")
	allExcludes, err := buildExclusions(logger)
	if err != nil {
		logger.Errorf("Error building exclusions: %v", err)
		return err
	}

	var paths []string
	if err := list.ListPaths(logger, rootfs, allExcludes, &paths); err != nil {
		logger.Errorf("Error listing paths: %v", err)
		return err
	}

	// Check if the file already exists.
	if _, err := os.Stat(file); err == nil {
		logger.Warnf("File %s already exists", file)
	} else if !os.IsNotExist(err) {
		logger.Errorf("Error checking file %s: %v", file, err)
		return fmt.Errorf("error checking file %s: %v", file, err)
	} else if os.IsNotExist(err) {
		// Save all environment variables to the file
		logger.Debugf("Saving environment variables to %s", dotenv)
		if err := env.SaveToFile(os.Environ(), dotenv); err != nil {
			logger.Errorf("Error saving environment variables %s: %v", dotenv, err)
			return fmt.Errorf("error saving environment variables %s: %v", dotenv, err)
		}
		// Create the tar archive
		logger.Debugf("Creating tar archive: %s", file)
		if err := archiver.Tar(logger, paths, file); err != nil {
			return err
		}
		logger.Infoln("Snapshot has created!")
	}
	return nil
}

// restoreCmd extracts the contents of the tar archive to the rootfs
// directory, while skipping directories listed in buildExclusions.
func restoreCmd(logger *logrus.Logger, file string) error {
	logger.Debugf("Restoring from archive: %s", file)
	allExcludes, err := buildExclusions(logger)
	if err != nil {
		logger.Errorf("Error building exclusions: %v", err)
		return err
	}

	if err := archiver.Untar(logger, file, rootfs, allExcludes); err != nil {
		return err
	}
	logger.Infoln("FS has restored!")
	return nil
}

// cleanupCmd removes files and directories in the rootfs directory,
// excluding the paths specified in buildExclusions.
func cleanupCmd(logger *logrus.Logger) error {
	logger.Debugf("Starting cleanup...")
	allExcludes, err := buildExclusions(logger)
	if err != nil {
		logger.Errorf("Error building exclusions: %v", err)
		return err
	}

	var paths []string
	if err := list.ListPaths(logger, rootfs, allExcludes, &paths); err != nil {
		logger.Errorf("Error listing paths for cleanup: %v", err)
		return err
	}


	logger.Debugf("Removing paths: %v", paths)
	if err := util.Rmrf(paths); err != nil {
		return err
	}
	logger.Infoln("Cleanup has completed!")
	return nil
}

func downloadCmd(logger *logrus.Logger, img string) error {
	logger.Debugf("Starting download of image: %s", img)
	tarName := "image.tar"
	envName := "image.env"
	diffEnvName := "old.env"

	dir := filepath.Join(workDir, util.Slugify(img))
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		logger.Errorf("Error creating directory %s: %v", dir, err)
		return fmt.Errorf("error creating directory %s: %v", dir, err)
	}

	tarFile := filepath.Join(dir, tarName)
	logger.Debugf("Getting filesystem of image %s and saving to %s", img, tarFile)
	if err := image.GetFS(logger, img, tarFile); err != nil {
		logger.Errorf("Error getting filesystem of image %s: %v", img, err)
		return fmt.Errorf("error getting image %s: %v", img, err)
	}

	logger.Debugf("Fetching environment variables for image %s", img)
	imgEnv, err := image.GetEnv(logger, img)
	if err != nil {
		logger.Errorf("Error getting environment variables for image %s: %v", img, err)
		return fmt.Errorf("error getting environment variables: %v", err)
	}

	dotenvFile := filepath.Join(dir, envName)
	logger.Debugf("Saving environment variables to %s", dotenvFile)
	if err := env.SaveToFile(imgEnv, dotenvFile); err != nil {
		logger.Errorf("Error saving environment variables to %s: %v", dotenvFile, err)
		return fmt.Errorf("error saving dotenv file %s: %v", dotenvFile, err)
	}

	diffEnv := env.DiffX(os.Environ(), imgEnv)
	diffDotEnvFile := filepath.Join(dir, diffEnvName)
	logger.Debugf("Saving environment diff to %s", diffDotEnvFile)
	if err := env.SaveToFile(diffEnv, diffDotEnvFile); err != nil {
		logger.Errorf("Error saving environment diff to %s: %v", diffDotEnvFile, err)
		return fmt.Errorf("error saving dotenv file %s: %v", diffDotEnvFile, err)
	}

	logger.Infof("Image %s has been downloaded!", img)
	return nil
}

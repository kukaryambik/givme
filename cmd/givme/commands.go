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
)

// buildExclusions generates a list of directories that should be excluded
// from operations such as snapshot creation or restoration.
func buildExclusions() ([]string, error) {
	mounts, err := util.GetMounts() // Get system mount points.
	if err != nil {
		return nil, err
	}

	// Append system directories and user-defined paths to exclusions.
	exclusions := append(mounts, "/proc", "/sys", "/dev", "/run",
		"/busybox", workDir, file, ".")

	// Add user-specified excluded directories from environment variable.
	if exclude != "" {
		exclusions = append(exclusions, strings.FieldsFunc(
			exclude, func(r rune) bool {
				return r == ':' || r == ','
			})...)
	}

	return exclusions, nil
}

// snapshotCmd creates a tar archive of the rootfs directory, excluding
// the directories specified in buildExclusions.
func snapshotCmd() error {
	allExcludes, err := buildExclusions()
	if err != nil {
		return err
	}

	var paths []string
	if err := list.ListPaths(rootfs, allExcludes, &paths); err != nil {
		return err
	}

	// Check if the file already exists.
	if _, err := os.Stat(file); err == nil {
		fmt.Fprintf(os.Stderr, "file %s already exists\n", file)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking file %s: %v", file, err)
	} else if os.IsNotExist(err) {
		// Save all environment variables to the file
		if err := env.SaveToFile(os.Environ(), dotenv); err != nil {
			return fmt.Errorf("error saving environment variables %s: %v", dotenv, err)
		}
		// Create the tar archive
		return archiver.Tar(paths, file)
	}
	return nil
}

// restoreCmd extracts the contents of the tar archive to the rootfs
// directory, while skipping directories listed in buildExclusions.
func restoreCmd(file string) error {
	allExcludes, err := buildExclusions()
	if err != nil {
		return err
	}

	return archiver.Untar(file, rootfs, allExcludes)
}

// cleanupCmd removes files and directories in the rootfs directory,
// excluding the paths specified in buildExclusions.
func cleanupCmd() error {
	allExcludes, err := buildExclusions()
	if err != nil {
		return err
	}

	var paths []string
	if err := list.ListPaths(rootfs, allExcludes, &paths); err != nil {
		return err
	}

	return util.Rmrf(paths)
}

func downloadCmd(img string) error {
	tarName := "image.tar"
	envName := "image.env"
	diffEnvName := "old.env"

	dir := filepath.Join(workDir, util.Slugify(img))
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory %s: %v", dir, err)
	}

	tarFile := filepath.Join(dir, tarName)
	if err := image.GetFS(img, tarFile); err != nil {
		return fmt.Errorf("error getting image %s: %v", img, err)
	}

	imgEnv, err := image.GetEnv(img)
	if err != nil {
		return fmt.Errorf("error getting environment variables: %v", err)
	}

	dotenvFile := filepath.Join(dir, envName)
	if err := env.SaveToFile(imgEnv, dotenvFile); err != nil {
		return fmt.Errorf("error saving dotenv file %s: %v", dotenvFile, err)
	}

	diffEnv := env.DiffX(os.Environ(), imgEnv)

	diffDotEnvFile := filepath.Join(dir, diffEnvName)
	if err := env.SaveToFile(diffEnv, diffDotEnvFile); err != nil {
		return fmt.Errorf("error saving dotenv file %s: %v", diffDotEnvFile, err)
	}

	return nil
}

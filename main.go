package main

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	c_ISUID = 04000 // setuid bit
	c_ISGID = 02000 // setgid bit
	c_ISVTX = 01000 // sticky bit
)

// getMountedDirs returns a list of mounted directories.
func getMountedDirs() ([]string, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dirs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			dirs = append(dirs, fields[1])
		}
	}
	return dirs, scanner.Err()
}

// isExcluded checks if a path is excluded.
func isExcluded(path string, excludedDirs []string) bool {
	if path == "/" {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	for _, exclude := range excludedDirs {
		if absPath == exclude || strings.HasPrefix(absPath, exclude+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// isDirEmpty checks if a directory is empty.
func isDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// addToArchive adds a file or directory to the archive, preserving essential attributes.
func addToArchive(path, source string, tarWriter *tar.Writer) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}

	relPath, err := filepath.Rel(source, path)
	if err != nil {
		return err
	}

	// Create the header manually
	header := &tar.Header{
		Name:       relPath,
		Mode:       int64(fi.Mode().Perm()),
		Size:       fi.Size(),
		ModTime:    fi.ModTime(),
		AccessTime: fi.ModTime(),
		ChangeTime: fi.ModTime(),
		Uid:        0,
		Gid:        0,
	}

	// Get owner and group
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		header.Uid = int(stat.Uid)
		header.Gid = int(stat.Gid)
	}

	// Preserve special bits
	header.Mode = int64(fi.Mode() & os.ModePerm)
	if fi.Mode()&os.ModeSetuid != 0 {
		header.Mode |= c_ISUID
	}
	if fi.Mode()&os.ModeSetgid != 0 {
		header.Mode |= c_ISGID
	}
	if fi.Mode()&os.ModeSticky != 0 {
		header.Mode |= c_ISVTX
	}

	// Determine file type and handle only regular files, directories, and symbolic links
	switch {
	case fi.Mode().IsRegular():
		header.Typeflag = tar.TypeReg
	case fi.Mode().IsDir():
		header.Typeflag = tar.TypeDir
	case fi.Mode()&os.ModeSymlink != 0:
		header.Typeflag = tar.TypeSymlink
		linkTarget, err := os.Readlink(path)
		if err != nil {
			return err
		}
		header.Linkname = linkTarget
	default:
		// Skip devices and pipes
		fmt.Printf("Skipped unsupported file type: %s\n", path)
		return nil
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	// If it's a regular file, write its content
	if fi.Mode().IsRegular() {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := io.Copy(tarWriter, f); err != nil {
			return err
		}
	}

	return nil
}

// archiveAndDeleteRecursively archives and deletes files and directories recursively, excluding specified paths.
func archiveAndDeleteRecursively(path, source string, tarWriter *tar.Writer, excludedDirs []string) error {
	// Check if the path is excluded.
	if isExcluded(path, excludedDirs) {
		return nil // Skip the excluded path
	}

	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if fi.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			err := archiveAndDeleteRecursively(filepath.Join(path, entry.Name()), source, tarWriter, excludedDirs)
			if err != nil {
				return err
			}
		}

		isEmpty, err := isDirEmpty(path)
		if err != nil {
			return err
		}

		// After processing all files in the directory, archive and delete it
		if err := addToArchive(path, source, tarWriter); err != nil {
			return fmt.Errorf("error adding %s to archive: %v", path, err)
		}

		if isEmpty && path != "/" {
			if err := os.Remove(path); err != nil {
				fmt.Printf("Error removing directory %s: %v\n", path, err)
			}
		}
	} else {
		if err := addToArchive(path, source, tarWriter); err != nil {
			return fmt.Errorf("error adding %s to archive: %v", path, err)
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			if err := os.Remove(path); err != nil {
				fmt.Printf("Error removing file %s: %v\n", path, err)
			}
		}
	}

	return nil
}

// restoreFromArchive extracts the archive to the specified directory, merging existing directories.
func restoreFromArchive(source, target string, excludedDirs []string) error {
	archivePath := filepath.Join(target, "snapshot.tar")

	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %v", err)
	}
	defer archiveFile.Close()

	tarReader := tar.NewReader(archiveFile)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // end of archive
		}
		if err != nil {
			return fmt.Errorf("error reading archive: %v", err)
		}

		path := filepath.Join(source, header.Name)

		// Check if the path is excluded.
		if isExcluded(path, excludedDirs) {
			fmt.Printf("Skipped excluded path during restoration: %s\n", path)
			// Skip reading the file content
			if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
				if _, err := io.Copy(io.Discard, tarReader); err != nil {
					return fmt.Errorf("error skipping file content %s: %v", header.Name, err)
				}
			}
			continue
		}

		// Remove existing files, but not directories
		if header.Typeflag != tar.TypeDir {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				fmt.Printf("Failed to remove existing file %s: %v\n", path, err)
			}
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create the directory if it does not exist
			if err := os.MkdirAll(path, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("error creating directory %s: %v", path, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			// Create all parent directories
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return fmt.Errorf("error creating directories %s: %v", filepath.Dir(path), err)
			}
			// Create the file with exact permissions
			outFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("error creating file %s: %v", path, err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("error writing to file %s: %v", path, err)
			}
			outFile.Close()
		case tar.TypeSymlink:
			// Create all parent directories
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return fmt.Errorf("error creating directories %s: %v", filepath.Dir(path), err)
			}
			// Create the symbolic link
			if err := os.Symlink(header.Linkname, path); err != nil {
				return fmt.Errorf("error creating symlink %s: %v", path, err)
			}
		default:
			fmt.Printf("Skipped unsupported file type during restoration: %s\n", header.Name)
		}

		// Set timestamps
		aTime := header.AccessTime
		if aTime.IsZero() {
			aTime = header.ModTime
		}
		if err := os.Chtimes(path, aTime, header.ModTime); err != nil {
			fmt.Printf("Failed to set timestamps for %s: %v\n", path, err)
		}

		// Set owner and group
		if err := os.Lchown(path, header.Uid, header.Gid); err != nil {
			fmt.Printf("Failed to set owner for %s: %v\n", path, err)
		}
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: program [snapshot|restore]")
		os.Exit(1)
	}

	command := os.Args[1]

	// Get workDir from environment variable or use the program's directory.
	exePath, err := os.Executable()
	if err != nil {
		exePath = "."
	}
	workDir := os.Getenv("GIVME_WORKDIR")
	if workDir == "" {
		workDir = filepath.Dir(exePath)
	}

	// Check if workDir exists and create it if necessary.
	if err := os.MkdirAll(workDir, 0755); err != nil {
		fmt.Printf("Error creating work directory: %v\n", err)
		return
	}

	// Set source from environment variable or use "/".
	source := os.Getenv("GIVME_SOURCE")
	if source == "" {
		source = "/"
	}

	target := filepath.Join(workDir, "snapshot.tar")

	// Get the list of mounted directories.
	mountedDirs, err := getMountedDirs()
	if err != nil {
		fmt.Printf("Error retrieving mounted directories: %v\n", err)
		return
	}

	// Initialize excluded directories.
	excludedDirs := append(
		mountedDirs,
		"/proc", "/sys", "/dev", "/run", // System directories
		workDir, // Add workDir to exclusions
	)

	// Add additional exclusions from environment variable.
	if additionalExclusions := os.Getenv("GIVME_EXCLUDE"); additionalExclusions != "" {
		exclusions := strings.FieldsFunc(additionalExclusions, func(r rune) bool {
			return r == ':' || r == ','
		})
		excludedDirs = append(excludedDirs, exclusions...)
	}

	// Convert exclusion paths to absolute paths
	for i, dir := range excludedDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			fmt.Printf("Error processing excluded directory %s: %v\n", dir, err)
			os.Exit(1)
		}
		excludedDirs[i] = absDir
	}

	switch command {
	case "snapshot":
		// Check if the target file exists.
		if _, err := os.Stat(target); err == nil {
			fmt.Printf("Error: file %s already exists.\n", target)
			os.Exit(1)
		} else if !os.IsNotExist(err) {
			fmt.Printf("Error checking file %s: %v\n", target, err)
			os.Exit(1)
		}

		tarFile, err := os.Create(target)
		if err != nil {
			fmt.Printf("Error creating archive file: %v\n", err)
			return
		}
		defer tarFile.Close()

		tarWriter := tar.NewWriter(tarFile)
		defer tarWriter.Close()

		if err := archiveAndDeleteRecursively(source, source, tarWriter, excludedDirs); err != nil {
			fmt.Printf("Error during archiving and deletion: %v\n", err)
		} else {
			fmt.Println("Archiving and deletion completed successfully.")
		}

	case "restore":
		if _, err := os.Stat(target); os.IsNotExist(err) {
			fmt.Printf("Error: archive file %s does not exist.\n", target)
			os.Exit(1)
		}

		if err := restoreFromArchive(source, workDir, excludedDirs); err != nil {
			fmt.Printf("Error restoring from archive: %v\n", err)
		} else {
			fmt.Println("Restoration completed successfully.")
		}

	default:
		fmt.Println("Unknown command. Use 'snapshot' or 'restore'.")
		os.Exit(1)
	}
}

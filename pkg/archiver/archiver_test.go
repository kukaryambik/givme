package archiver

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// Helper function to compare two directories recursively
func directoriesEqual(dir1, dir2 string) (bool, error) {
	var dir1Files []string
	var dir2Files []string

	err := filepath.Walk(dir1, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(dir1, path)
		if err != nil {
			return err
		}
		dir1Files = append(dir1Files, relPath)
		return nil
	})
	if err != nil {
		return false, err
	}

	err = filepath.Walk(dir2, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(dir2, path)
		if err != nil {
			return err
		}
		dir2Files = append(dir2Files, relPath)
		return nil
	})
	if err != nil {
		return false, err
	}

	if len(dir1Files) != len(dir2Files) {
		return false, nil
	}

	for i := range dir1Files {
		if dir1Files[i] != dir2Files[i] {
			return false, nil
		}
	}

	return true, nil
}

func TestRegularFiles(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create regular files
	err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("Hello World"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Archive
	tarPath := filepath.Join(dstDir, "archive.tar")
	err = Tar(srcDir, tarPath, nil)
	if err != nil {
		t.Fatalf("Tar failed: %v", err)
	}

	// Extract
	tar, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("Failed to open tar archive: %v", err)
	}
	defer tar.Close()
	extractDir := filepath.Join(dstDir, "extracted")
	err = Untar(tar, extractDir, nil)
	if err != nil {
		t.Fatalf("Untar failed: %v", err)
	}

	// Assert
	extractedFile := filepath.Join(extractDir, "file1.txt")
	if _, err := os.Stat(extractedFile); os.IsNotExist(err) {
		t.Errorf("Expected file does not exist: %s", extractedFile)
	}

	content, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}
	if string(content) != "Hello World" {
		t.Errorf("Content mismatch: expected 'Hello World', got '%s'", content)
	}
}

func TestDirectories(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create nested directories and files
	nestedDir := filepath.Join(srcDir, "dir1", "dir2")
	err := os.MkdirAll(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested directories: %v", err)
	}
	err = os.WriteFile(filepath.Join(nestedDir, "file.txt"), []byte("Nested"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file in nested directory: %v", err)
	}

	// Archive
	tarPath := filepath.Join(dstDir, "archive.tar")
	err = Tar(srcDir, tarPath, nil)
	if err != nil {
		t.Fatalf("Tar failed: %v", err)
	}

	// Extract
	tar, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("Failed to open tar archive: %v", err)
	}
	defer tar.Close()
	extractDir := filepath.Join(dstDir, "extracted")
	err = Untar(tar, extractDir, nil)
	if err != nil {
		t.Fatalf("Untar failed: %v", err)
	}

	// Assert
	extractedFile := filepath.Join(extractDir, "dir1", "dir2", "file.txt")
	if _, err := os.Stat(extractedFile); os.IsNotExist(err) {
		t.Errorf("Expected file does not exist: %s", extractedFile)
	}

	// Compare directories
	equal, err := directoriesEqual(srcDir, extractDir)
	if err != nil {
		t.Fatalf("Failed to compare directories: %v", err)
	}
	if !equal {
		t.Errorf("Directory structures do not match")
	}
}

func TestSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping symlink test on Windows")
	}

	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a file and a symlink to it
	targetFile := filepath.Join(srcDir, "target.txt")
	err := os.WriteFile(targetFile, []byte("Target"), 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}
	symlink := filepath.Join(srcDir, "symlink.txt")
	err = os.Symlink("target.txt", symlink)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Archive
	tarPath := filepath.Join(dstDir, "archive.tar")
	err = Tar(srcDir, tarPath, nil)
	if err != nil {
		t.Fatalf("Tar failed: %v", err)
	}

	// Extract
	tar, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("Failed to open tar archive: %v", err)
	}
	defer tar.Close()
	extractDir := filepath.Join(dstDir, "extracted")
	err = Untar(tar, extractDir, nil)
	if err != nil {
		t.Fatalf("Untar failed: %v", err)
	}

	// Assert
	extractedSymlink := filepath.Join(extractDir, "symlink.txt")
	linkTarget, err := os.Readlink(extractedSymlink)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}
	if linkTarget != "target.txt" {
		t.Errorf("Symlink target mismatch: expected 'target.txt', got '%s'", linkTarget)
	}
}

func TestHardLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping hard link test on Windows")
	}

	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a file
	originalFile := filepath.Join(srcDir, "original.txt")
	err := os.WriteFile(originalFile, []byte("Original"), 0644)
	if err != nil {
		t.Fatalf("Failed to create original file: %v", err)
	}

	// Create a hard link
	hardLink := filepath.Join(srcDir, "hardlink.txt")
	err = os.Link(originalFile, hardLink)
	if err != nil {
		t.Fatalf("Failed to create hard link: %v", err)
	}

	// Archive
	tarPath := filepath.Join(dstDir, "archive.tar")
	err = Tar(srcDir, tarPath, nil)
	if err != nil {
		t.Fatalf("Tar failed: %v", err)
	}

	// Extract
	tar, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("Failed to open tar archive: %v", err)
	}
	defer tar.Close()
	extractDir := filepath.Join(dstDir, "extracted")
	err = Untar(tar, extractDir, nil)
	if err != nil {
		t.Fatalf("Untar failed: %v", err)
	}

	// Assert
	extractedOriginal := filepath.Join(extractDir, "original.txt")
	extractedHardLink := filepath.Join(extractDir, "hardlink.txt")

	originalInfo, err := os.Stat(extractedOriginal)
	if err != nil {
		t.Fatalf("Failed to stat original file: %v", err)
	}
	hardLinkInfo, err := os.Stat(extractedHardLink)
	if err != nil {
		t.Fatalf("Failed to stat hard link: %v", err)
	}

	originalSys, ok1 := originalInfo.Sys().(*unix.Stat_t)
	hardLinkSys, ok2 := hardLinkInfo.Sys().(*unix.Stat_t)
	if !ok1 || !ok2 {
		t.Fatalf("Failed to get raw unix.Stat_t data")
	}

	if originalSys.Ino != hardLinkSys.Ino {
		t.Errorf("Hard links do not point to the same inode")
	}
}

func TestExclusions(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create files
	includeFile := filepath.Join(srcDir, "include.txt")
	excludeFile := filepath.Join(srcDir, "exclude.txt")
	err := os.WriteFile(includeFile, []byte("Include"), 0644)
	if err != nil {
		t.Fatalf("Failed to create include file: %v", err)
	}
	err = os.WriteFile(excludeFile, []byte("Exclude"), 0644)
	if err != nil {
		t.Fatalf("Failed to create exclude file: %v", err)
	}

	// Archive with exclusions
	tarPath := filepath.Join(dstDir, "archive.tar")
	err = Tar(srcDir, tarPath, []string{excludeFile})
	if err != nil {
		t.Fatalf("Tar failed: %v", err)
	}

	// Extract
	tar, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("Failed to open tar archive: %v", err)
	}
	defer tar.Close()
	extractDir := filepath.Join(dstDir, "extracted")
	err = Untar(tar, extractDir, nil)
	if err != nil {
		t.Fatalf("Untar failed: %v", err)
	}

	// Assert
	if _, err := os.Stat(filepath.Join(extractDir, "include.txt")); os.IsNotExist(err) {
		t.Errorf("Included file is missing")
	}
	if _, err := os.Stat(filepath.Join(extractDir, "exclude.txt")); err == nil {
		t.Errorf("Excluded file was found in the archive")
	}
}

func TestPermissions(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a file with specific permissions
	filePath := filepath.Join(srcDir, "file.txt")
	err := os.WriteFile(filePath, []byte("Content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Archive
	tarPath := filepath.Join(dstDir, "archive.tar")
	err = Tar(srcDir, tarPath, nil)
	if err != nil {
		t.Fatalf("Tar failed: %v", err)
	}

	// Extract
	tar, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("Failed to open tar archive: %v", err)
	}
	defer tar.Close()
	extractDir := filepath.Join(dstDir, "extracted")
	err = Untar(tar, extractDir, nil)
	if err != nil {
		t.Fatalf("Untar failed: %v", err)
	}

	// Assert
	extractedFile := filepath.Join(extractDir, "file.txt")
	info, err := os.Stat(extractedFile)
	if err != nil {
		t.Fatalf("Failed to stat extracted file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("Permissions mismatch: expected 0600, got %o", info.Mode().Perm())
	}
}

func TestOwnership(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Skipping ownership test; not running as root")
	}

	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a file
	filePath := filepath.Join(srcDir, "file.txt")
	err := os.WriteFile(filePath, []byte("Content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Change ownership
	err = os.Chown(filePath, 1000, 1000) // Assuming UID and GID 1000 exist
	if err != nil {
		t.Fatalf("Failed to change ownership: %v", err)
	}

	// Archive
	tarPath := filepath.Join(dstDir, "archive.tar")
	err = Tar(srcDir, tarPath, nil)
	if err != nil {
		t.Fatalf("Tar failed: %v", err)
	}

	// Extract
	tar, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("Failed to open tar archive: %v", err)
	}
	defer tar.Close()
	extractDir := filepath.Join(dstDir, "extracted")
	err = Untar(tar, extractDir, nil)
	if err != nil {
		t.Fatalf("Untar failed: %v", err)
	}

	// Assert
	extractedFile := filepath.Join(extractDir, "file.txt")
	info, err := os.Stat(extractedFile)
	if err != nil {
		t.Fatalf("Failed to stat extracted file: %v", err)
	}

	stat, ok := info.Sys().(*unix.Stat_t)
	if !ok {
		t.Fatalf("Failed to get raw unix.Stat_t data")
	}

	if int(stat.Uid) != 1000 || int(stat.Gid) != 1000 {
		t.Errorf("Ownership mismatch: expected UID and GID 1000, got UID %d, GID %d", stat.Uid, stat.Gid)
	}
}

func TestTimes(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a file
	filePath := filepath.Join(srcDir, "file.txt")
	err := os.WriteFile(filePath, []byte("Content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Set specific modification time
	mtime := time.Unix(1600000000, 0)
	err = os.Chtimes(filePath, time.Now(), mtime) // atime is set to now, mtime to specific time
	if err != nil {
		t.Fatalf("Failed to change times: %v", err)
	}

	// Archive
	tarPath := filepath.Join(dstDir, "archive.tar")
	err = Tar(srcDir, tarPath, nil)
	if err != nil {
		t.Fatalf("Tar failed: %v", err)
	}

	// Extract
	tar, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("Failed to open tar archive: %v", err)
	}
	defer tar.Close()
	extractDir := filepath.Join(dstDir, "extracted")
	err = Untar(tar, extractDir, nil)
	if err != nil {
		t.Fatalf("Untar failed: %v", err)
	}

	// Assert
	extractedFile := filepath.Join(extractDir, "file.txt")
	info, err := os.Stat(extractedFile)
	if err != nil {
		t.Fatalf("Failed to stat extracted file: %v", err)
	}

	// Check modification time
	if !info.ModTime().Equal(mtime) {
		t.Errorf("Modification time mismatch: expected %v, got %v", mtime, info.ModTime())
	}
}

func TestEmptyFilesAndDirectories(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create empty file and directory
	emptyFile := filepath.Join(srcDir, "empty_file.txt")
	err := os.WriteFile(emptyFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	emptyDir := filepath.Join(srcDir, "empty_dir")
	err = os.Mkdir(emptyDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}

	// Archive
	tarPath := filepath.Join(dstDir, "archive.tar")
	err = Tar(srcDir, tarPath, nil)
	if err != nil {
		t.Fatalf("Tar failed: %v", err)
	}

	// Extract
	tar, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("Failed to open tar archive: %v", err)
	}
	defer tar.Close()
	extractDir := filepath.Join(dstDir, "extracted")
	err = Untar(tar, extractDir, nil)
	if err != nil {
		t.Fatalf("Untar failed: %v", err)
	}

	// Assert empty file exists
	if info, err := os.Stat(filepath.Join(extractDir, "empty_file.txt")); err != nil || info.Size() != 0 {
		t.Errorf("Empty file was not correctly archived and extracted")
	}

	// Assert empty directory exists
	if info, err := os.Stat(filepath.Join(extractDir, "empty_dir")); err != nil || !info.IsDir() {
		t.Errorf("Empty directory was not correctly archived and extracted")
	}
}

func TestLongPaths(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a long path
	longDir := srcDir
	for i := 0; i < 10; i++ {
		longDir = filepath.Join(longDir, "very_long_directory_name_to_test_the_limits_of_the_file_system")
	}
	err := os.MkdirAll(longDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create long path: %v", err)
	}

	// Create a file in the long path
	longFile := filepath.Join(longDir, "file.txt")
	err = os.WriteFile(longFile, []byte("Long Path"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file in long path: %v", err)
	}

	// Archive
	tarPath := filepath.Join(dstDir, "archive.tar")
	err = Tar(srcDir, tarPath, nil)
	if err != nil {
		t.Fatalf("Tar failed: %v", err)
	}

	// Extract
	tar, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("Failed to open tar archive: %v", err)
	}
	defer tar.Close()
	extractDir := filepath.Join(dstDir, "extracted")
	err = Untar(tar, extractDir, nil)
	if err != nil {
		t.Fatalf("Untar failed: %v", err)
	}

	// Assert
	rel, err := filepath.Rel(srcDir, longFile)
	if err != nil {
		t.Fatalf("Failed to get relative path: %v", err)
	}
	extractedFile := filepath.Join(extractDir, rel)
	if _, err := os.Stat(extractedFile); os.IsNotExist(err) {
		t.Errorf("File with long path does not exist: %s", extractedFile)
	}
}

func TestArchiveAndExtract(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create various files and directories
	err := os.MkdirAll(filepath.Join(srcDir, "dir", "subdir"), 0755)
	if err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}
	err = os.WriteFile(filepath.Join(srcDir, "dir", "subdir", "file.txt"), []byte("Content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create symlink
	if runtime.GOOS != "windows" {
		err = os.Symlink("file.txt", filepath.Join(srcDir, "dir", "subdir", "symlink.txt"))
		if err != nil {
			t.Fatalf("Failed to create symlink: %v", err)
		}
	}

	// Archive
	tarPath := filepath.Join(dstDir, "archive.tar")
	err = Tar(srcDir, tarPath, nil)
	if err != nil {
		t.Fatalf("Tar failed: %v", err)
	}

	// Extract
	tar, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("Failed to open tar archive: %v", err)
	}
	defer tar.Close()
	extractDir := filepath.Join(dstDir, "extracted")
	err = Untar(tar, extractDir, nil)
	if err != nil {
		t.Fatalf("Untar failed: %v", err)
	}

	// Compare directories
	equal, err := directoriesEqual(srcDir, extractDir)
	if err != nil {
		t.Fatalf("Failed to compare directories: %v", err)
	}
	if !equal {
		t.Errorf("Directory structures do not match")
	}
}

func TestSpecialFiles(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("Skipping special files test; not running as root")
	}

	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a named pipe (FIFO)
	pipePath := filepath.Join(srcDir, "mypipe")
	err := unix.Mkfifo(pipePath, 0644)
	if err != nil {
		t.Fatalf("Failed to create FIFO: %v", err)
	}

	// Archive
	tarPath := filepath.Join(dstDir, "archive.tar")
	err = Tar(srcDir, tarPath, nil)
	if err != nil {
		t.Fatalf("Tar failed: %v", err)
	}

	// Extract
	tar, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("Failed to open tar archive: %v", err)
	}
	defer tar.Close()
	extractDir := filepath.Join(dstDir, "extracted")
	err = Untar(tar, extractDir, nil)
	if err != nil {
		t.Fatalf("Untar failed: %v", err)
	}

	// Assert
	extractedPipe := filepath.Join(extractDir, "mypipe")
	info, err := os.Stat(extractedPipe)
	if err != nil {
		t.Fatalf("Failed to stat extracted FIFO: %v", err)
	}

	if info.Mode()&os.ModeNamedPipe == 0 {
		t.Errorf("Extracted file is not a FIFO as expected")
	}
}

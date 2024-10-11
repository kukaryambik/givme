package archiver_test

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	// Replace with the correct import path of your archiver package
	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/stretchr/testify/assert"
)

// TestTarAndUntar tests the basic functionality of archiving and unarchiving files and directories.
func TestTarAndUntar(t *testing.T) {
	// Create a temporary source directory with files and subdirectories.
	srcDir, err := os.MkdirTemp("", "archiver_test_src")
	assert.NoError(t, err)
	defer os.RemoveAll(srcDir)

	// Create a subdirectory.
	err = os.Mkdir(filepath.Join(srcDir, "subdir"), 0755)
	assert.NoError(t, err)

	// Create files in the source directory and subdirectory.
	err = os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("This is file 1"), 0644)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("This is file 2"), 0644)
	assert.NoError(t, err)

	// Create a symbolic link in the source directory.
	err = os.Symlink("file1.txt", filepath.Join(srcDir, "link_to_file1"))
	assert.NoError(t, err)

	// Create a temporary file for the archive.
	archiveFile, err := os.CreateTemp("", "archiver_test_archive.tar")
	assert.NoError(t, err)
	defer os.Remove(archiveFile.Name())
	archiveFile.Close() // Close the file; Tar will open it.

	// Archive the source directory.
	err = archiver.Tar(srcDir, archiveFile.Name(), nil)
	assert.NoError(t, err)

	// Create a temporary destination directory for extraction.
	dstDir, err := os.MkdirTemp("", "archiver_test_dst")
	assert.NoError(t, err)
	defer os.RemoveAll(dstDir)

	// Unarchive into the destination directory.
	err = archiver.Untar(archiveFile.Name(), dstDir, nil)
	assert.NoError(t, err)

	// Verify that the files exist and have correct contents.
	content, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "This is file 1", string(content))

	content, err = os.ReadFile(filepath.Join(dstDir, "subdir", "file2.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "This is file 2", string(content))

	// Verify that the symbolic link exists and points to the correct file.
	linkTarget, err := os.Readlink(filepath.Join(dstDir, "link_to_file1"))
	assert.NoError(t, err)
	assert.Equal(t, "file1.txt", linkTarget)
}

// TestTarWithExclusions tests archiving with specific files excluded.
func TestTarWithExclusions(t *testing.T) {
	// Create a temporary source directory.
	srcDir, err := os.MkdirTemp("", "archiver_test_src")
	assert.NoError(t, err)
	defer os.RemoveAll(srcDir)

	// Create files in the source directory.
	err = os.WriteFile(filepath.Join(srcDir, "include.txt"), []byte("Include this file"), 0644)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(srcDir, "exclude.txt"), []byte("Exclude this file"), 0644)
	assert.NoError(t, err)

	// Create a temporary file for the archive.
	archiveFile, err := os.CreateTemp("", "archiver_test_archive.tar")
	assert.NoError(t, err)
	defer os.Remove(archiveFile.Name())
	archiveFile.Close()

	// Archive the source directory, excluding "exclude.txt".
	exclusions := []string{filepath.Join(srcDir, "exclude.txt")}
	err = archiver.Tar(srcDir, archiveFile.Name(), exclusions)
	assert.NoError(t, err)

	// Open the archive and verify contents.
	input, err := os.Open(archiveFile.Name())
	assert.NoError(t, err)
	defer input.Close()

	tarReader := tar.NewReader(input)
	foundInclude := false
	foundExclude := false

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive.
		}
		assert.NoError(t, err)

		if hdr.Name == "include.txt" {
			foundInclude = true
		}
		if hdr.Name == "exclude.txt" {
			foundExclude = true
		}
	}

	assert.True(t, foundInclude, "include.txt should be in the archive")
	assert.False(t, foundExclude, "exclude.txt should not be in the archive")
}

// TestUntarWithExclusions tests unarchiving with specific files excluded.
func TestUntarWithExclusions(t *testing.T) {
	// Create an in-memory tar archive with known files.
	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)

	// Add "include.txt" to the archive.
	err := tarWriter.WriteHeader(&tar.Header{
		Name: "include.txt",
		Mode: 0644,
		Size: int64(len("Include this file")),
	})
	assert.NoError(t, err)
	_, err = tarWriter.Write([]byte("Include this file"))
	assert.NoError(t, err)

	// Add "exclude.txt" to the archive.
	err = tarWriter.WriteHeader(&tar.Header{
		Name: "exclude.txt",
		Mode: 0644,
		Size: int64(len("Exclude this file")),
	})
	assert.NoError(t, err)
	_, err = tarWriter.Write([]byte("Exclude this file"))
	assert.NoError(t, err)

	// Close the tar writer.
	err = tarWriter.Close()
	assert.NoError(t, err)

	// Write the tar archive to a temporary file.
	archiveFile, err := os.CreateTemp("", "archiver_test_archive.tar")
	assert.NoError(t, err)
	defer os.Remove(archiveFile.Name())
	_, err = archiveFile.Write(buf.Bytes())
	assert.NoError(t, err)
	archiveFile.Close()

	// Create a temporary destination directory.
	dstDir, err := os.MkdirTemp("", "archiver_test_dst")
	assert.NoError(t, err)
	defer os.RemoveAll(dstDir)

	// Unarchive, excluding "exclude.txt".
	exclusions := []string{filepath.Join(dstDir, "exclude.txt")}
	err = archiver.Untar(archiveFile.Name(), dstDir, exclusions)
	assert.NoError(t, err)

	// Verify that "include.txt" exists.
	content, err := os.ReadFile(filepath.Join(dstDir, "include.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "Include this file", string(content))

	// Verify that "exclude.txt" does not exist.
	_, err = os.Stat(filepath.Join(dstDir, "exclude.txt"))
	assert.True(t, os.IsNotExist(err))
}

// TestSymlinks tests that symbolic links are correctly archived and unarchived.
func TestSymlinks(t *testing.T) {
	// Create a temporary source directory.
	srcDir, err := os.MkdirTemp("", "archiver_test_src")
	assert.NoError(t, err)
	defer os.RemoveAll(srcDir)

	// Create a file.
	err = os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("This is a file"), 0644)
	assert.NoError(t, err)

	// Create a symbolic link to the file.
	err = os.Symlink("file.txt", filepath.Join(srcDir, "link_to_file.txt"))
	assert.NoError(t, err)

	// Create a temporary file for the archive.
	archiveFile, err := os.CreateTemp("", "archiver_test_archive.tar")
	assert.NoError(t, err)
	defer os.Remove(archiveFile.Name())
	archiveFile.Close()

	// Archive the source directory.
	err = archiver.Tar(srcDir, archiveFile.Name(), nil)
	assert.NoError(t, err)

	// Create a temporary destination directory.
	dstDir, err := os.MkdirTemp("", "archiver_test_dst")
	assert.NoError(t, err)
	defer os.RemoveAll(dstDir)

	// Unarchive the archive.
	err = archiver.Untar(archiveFile.Name(), dstDir, nil)
	assert.NoError(t, err)

	// Verify that the symbolic link exists and points correctly.
	linkTarget, err := os.Readlink(filepath.Join(dstDir, "link_to_file.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "file.txt", linkTarget)

	// Verify the linked file's content.
	content, err := os.ReadFile(filepath.Join(dstDir, "file.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "This is a file", string(content))
}

// TestEmptyDirectories tests that empty directories are correctly archived and unarchived.
func TestEmptyDirectories(t *testing.T) {
	// Create a temporary source directory.
	srcDir, err := os.MkdirTemp("", "archiver_test_src")
	assert.NoError(t, err)
	defer os.RemoveAll(srcDir)

	// Create an empty subdirectory.
	err = os.Mkdir(filepath.Join(srcDir, "empty_dir"), 0755)
	assert.NoError(t, err)

	// Create a temporary file for the archive.
	archiveFile, err := os.CreateTemp("", "archiver_test_archive.tar")
	assert.NoError(t, err)
	defer os.Remove(archiveFile.Name())
	archiveFile.Close()

	// Archive the source directory.
	err = archiver.Tar(srcDir, archiveFile.Name(), nil)
	assert.NoError(t, err)

	// Create a temporary destination directory.
	dstDir, err := os.MkdirTemp("", "archiver_test_dst")
	assert.NoError(t, err)
	defer os.RemoveAll(dstDir)

	// Unarchive the archive.
	err = archiver.Untar(archiveFile.Name(), dstDir, nil)
	assert.NoError(t, err)

	// Verify that the empty directory exists.
	info, err := os.Stat(filepath.Join(dstDir, "empty_dir"))
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify that the directory is empty.
	entries, err := os.ReadDir(filepath.Join(dstDir, "empty_dir"))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(entries))
}

// TestHardLinks tests that hard links are correctly archived and unarchived.
func TestHardLinks(t *testing.T) {
	// Create a temporary source directory.
	srcDir, err := os.MkdirTemp("", "archiver_test_src")
	assert.NoError(t, err)
	defer os.RemoveAll(srcDir)

	// Create a file.
	err = os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("This is a file"), 0644)
	assert.NoError(t, err)

	// Create a hard link to the file.
	err = os.Link(filepath.Join(srcDir, "file.txt"), filepath.Join(srcDir, "hard_link_to_file.txt"))
	assert.NoError(t, err)

	// Create a temporary file for the archive.
	archiveFile, err := os.CreateTemp("", "archiver_test_archive.tar")
	assert.NoError(t, err)
	defer os.Remove(archiveFile.Name())
	archiveFile.Close()

	// Archive the source directory.
	err = archiver.Tar(srcDir, archiveFile.Name(), nil)
	assert.NoError(t, err)

	// Create a temporary destination directory.
	dstDir, err := os.MkdirTemp("", "archiver_test_dst")
	assert.NoError(t, err)
	defer os.RemoveAll(dstDir)

	// Unarchive the archive.
	err = archiver.Untar(archiveFile.Name(), dstDir, nil)
	assert.NoError(t, err)

	// Verify that the hard link exists.
	infoFile, err := os.Stat(filepath.Join(dstDir, "file.txt"))
	os.Link(filepath.Join(dstDir, "file.txt"), filepath.Join(dstDir, "test_link_to_file.txt"))
	assert.NoError(t, err)

	infoHardLink, err := os.Stat(filepath.Join(dstDir, "hard_link_to_file.txt"))
	assert.NoError(t, err)

	// On Unix, os.SameFile can check if two files are the same inode.
	assert.True(t, os.SameFile(infoFile, infoHardLink), "Files should be hard links (same inode)")

	// Verify the content of the file via the hard link.
	content, err := os.ReadFile(filepath.Join(dstDir, "hard_link_to_file.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "This is a file", string(content))
}

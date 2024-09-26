package list

import (
	"os"
	"path/filepath"

	"github.com/kukaryambik/rumett/pkg/util"
)

// ListPaths walks through files and directories recursively, excluding specified paths.
func ListPaths(path string, exclude []string, list *[]string) error {
	p, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if util.IsPathFrom(p, exclude) {
		return nil
	}

	fi, err := os.Lstat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if fi.IsDir() {
		// Check if the path contains exclusions.
		if util.IsPathContains(p, exclude) {
			entries, err := os.ReadDir(p)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				f := filepath.Join(p, entry.Name())
				err := ListPaths(f, exclude, list)
				if err != nil {
					return err
				}
			}
		} else {
			*list = append(*list, p)
		}
	} else {
		*list = append(*list, p)
	}

	return nil
}

package exclusions

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/kukaryambik/givme/pkg/util"
)

var (
	// Default values for systemExcludes
	systemExclusions = []string{"/proc", "/sys", "/dev", "/run"}
)

// Build generates a list of paths that should be excluded
// from operations such as snapshot creation or restoration.
func Build(excl ...string) ([]string, error) {
	mounts, err := util.GetMounts() // Get system mount points.
	if err != nil {
		return nil, err
	}

	var others []string
	for _, e := range excl {
		s := strings.FieldsFunc(e, func(r rune) bool { return r == ':' || r == ',' })
		for _, d := range s {
			absPath, err := filepath.Abs(d)
			if err != nil {
				return nil, err
			}
			others = append(others, absPath)
		}
	}

	return slices.Concat(systemExclusions, mounts, others), nil
}

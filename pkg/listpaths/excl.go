package listpaths

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kukaryambik/givme/pkg/util"
)

var (
	// Default values for systemExcl
	systemExcl = []string{"/proc", "/sys", "/dev", "/run", "/var/run"}
)

// Excl generates a list of paths that should be excluded
// from operations such as snapshot creation or restoration.
func Excl(rootpath string, exclude []string) ([]string, error) {
	mounts, err := util.GetMounts() // Get system mount points.
	if err != nil {
		return nil, err
	}

	var userExcl []string
	var exclFromExcl []string
	for _, e := range exclude {
		s := strings.FieldsFunc(e, func(r rune) bool { return r == ':' || r == ',' })
		for _, d := range s {
			if strings.HasPrefix(d, "!") {
				exclFromExcl = append(exclFromExcl, d[1:])
			} else {
				userExcl = append(userExcl, d)
			}
		}
	}

	allExcl := slices.Concat(mounts, systemExcl, userExcl)
	allExcl, err = util.AbsAll(allExcl)
	if err != nil {
		return nil, err
	}
	allExcl = util.UniqString(allExcl)

	exclFromExcl, err = util.AbsAll(exclFromExcl)
	if err != nil {
		return nil, err
	}

	absRoot, err := filepath.Abs(rootpath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", rootpath, err)
	}

	var finalExcl []string
	for i, v := range allExcl {
		// Create local copy of slice without v
		a := slices.Clone(allExcl)
		a = slices.Delete(a, i, i+1)

		// Check if the path should be excluded
		if util.IsPathFrom(v, a) {
			continue
		}

		// Check if the path contain some excludes in it
		if err := List(absRoot, v, exclFromExcl, &finalExcl); err != nil {
			return nil, err
		}
	}

	return finalExcl, nil
}

package listpaths

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/kukaryambik/givme/pkg/util"
)

var (
	// Default values for systemExcl
	systemExcl = []string{"/proc", "/sys", "/dev", "/run", "/var/run"}
)

// Exclude generates a list of paths that should be excluded
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
				absPath, err := filepath.Abs(d[1:])
				if err != nil {
					return nil, err
				}
				exclFromExcl = append(exclFromExcl, absPath)
			} else {
				absPath, err := filepath.Abs(d)
				if err != nil {
					return nil, err
				}
				userExcl = append(userExcl, absPath)
			}
		}
	}

	allExcl := slices.Concat(mounts, systemExcl, userExcl)
	allExcl = util.UniqString(allExcl)

	var finalExcl []string
	for i, v := range allExcl {
		// Create local copy of slice without v
		a := slices.Clone(allExcl)
		a = slices.Delete(a, i, i+1)

		//
		if ok, _ := util.IsPathFrom(v, a); ok {
			continue
		}

		//
		if err := List(rootpath, v, exclFromExcl, &finalExcl); err != nil {
			return nil, err
		}
	}

	return finalExcl, nil
}

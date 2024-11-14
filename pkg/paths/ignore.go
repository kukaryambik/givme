package paths

import (
	"slices"
	"strings"

	"github.com/kukaryambik/givme/pkg/util"
)

type IgnoreConf struct {
	Exclusions    []string
	IgnoreExecDir bool
	IgnoreMounts  bool
	IgnorePaths   []string
	IgnoreSystem  bool
}

var (
	SystemDirs = []string{"/proc", "/sys", "/dev", "/run", "/var/run"}
)

func Ignore(i []string) *IgnoreConf {
	var excl, ignore []string
	for _, p := range i {
		if strings.HasPrefix(p, "!") {
			excl = append(excl, p[1:])
		} else {
			ignore = append(ignore, p)
		}
	}
	return &IgnoreConf{
		Exclusions:    excl,
		IgnoreExecDir: true,
		IgnoreMounts:  true,
		IgnoreSystem:  true,
		IgnorePaths:   ignore,
	}
}

func (conf *IgnoreConf) AddPaths(p ...string) *IgnoreConf {
	conf.IgnorePaths = append(conf.IgnorePaths, p...)
	return conf
}

func (conf *IgnoreConf) ExclFromList(p ...string) *IgnoreConf {
	conf.Exclusions = append(conf.Exclusions, p...)
	return conf
}

// List generates the list of paths that should be ignored
func (conf *IgnoreConf) List() ([]string, error) {
	if conf.IgnoreMounts {
		mounts, err := GetMounts() // Get system mount points.
		if err != nil {
			return nil, err
		}
		conf.IgnorePaths = append(conf.IgnorePaths, mounts...)
	}

	if conf.IgnoreExecDir {
		conf.IgnorePaths = append(conf.IgnorePaths, util.GetExecDir())
	}

	if conf.IgnoreSystem {
		conf.IgnorePaths = append(conf.IgnorePaths, SystemDirs...)
	}

	ignore, err := AbsAll(conf.IgnorePaths)
	if err != nil {
		return nil, err
	}

	excl, err := AbsAll(conf.Exclusions)
	if err != nil {
		return nil, err
	}

	ignore = util.UniqString(ignore)
	excl = util.UniqString(excl)

	var lst []string
	for i, v := range ignore {
		// Create local copy of slice without v
		a := slices.Clone(ignore)
		a = slices.Delete(a, i, i+1)

		// Check if the path should be ignored
		if PathFrom(v, a) {
			continue
		}

		// Check if the path contain some ignores in it
		if err := GetList(v, excl, &lst); err != nil {
			return nil, err
		}
	}

	return lst, nil
}

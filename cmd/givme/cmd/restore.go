package cmd

import (
	"fmt"
	"os"

	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
)

// Restore extracts the contents of the tar archive to the rootfs
// directory, while skipping directories listed in buildExclusions.
func restore(opts *CommandOptions) error {
	logrus.Debugf("Restoring from archive: %s", opts.TarFile)

	if err := cleanup(opts); err != nil {
		return err
	}

	// Configure ignored paths
	ignoreConf := paths.Ignore(opts.IgnorePaths).ExclFromList(opts.RootFS)
	ignores, err := ignoreConf.AddPaths(opts.Workdir).List()
	if err != nil {
		return err
	}

	if err := archiver.Untar(opts.TarFile, opts.RootFS, ignores); err != nil {
		return err
	}

	if opts.DotenvFile != "" && opts.Eval {
		f, err := os.ReadFile(opts.DotenvFile)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", string(f))
	}

	logrus.Infoln("FS has restored!")
	return nil
}

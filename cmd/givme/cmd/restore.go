package cmd

import (
	"fmt"
	"os"

	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/sirupsen/logrus"
)

// Restore extracts the contents of the tar archive to the rootfs
// directory, while skipping directories listed in buildExclusions.
func restore(conf *CommandOptions) error {
	logrus.Debugf("Restoring from archive: %s", conf.TarFile)

	if err := archiver.Untar(conf.TarFile, conf.RootFS, conf.Exclusions); err != nil {
		return err
	}

	if conf.DotenvFile != "" {
		f, err := os.ReadFile(conf.DotenvFile)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", string(f))
	}

	logrus.Infoln("FS has restored!")
	return nil
}

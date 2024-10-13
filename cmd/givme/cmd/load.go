package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func load(opts *CommandOptions) error {

	img, err := save(opts)
	if err != nil {
		return err
	}

	tmpFS := filepath.Join(opts.Workdir, ".fs_"+util.Slugify(opts.Image)+".tar")
	defer os.Remove(tmpFS)

	if err := img.Export(tmpFS); err != nil {
		return err
	}

	logrus.Debugf("Fetching config file for image %s", img.Name)
	cfg, err := img.Config()
	if err != nil {
		return fmt.Errorf("error getting config from image %s: %v", img, err)
	}

	if err := cleanup(opts); err != nil {
		return err
	}

	// if opts.Volumes {
	// 	for v := range cfg.Config.Volumes {
	// 		if err := os.MkdirAll(filepath.Dir(v), os.ModePerm); err != nil {
	// 			return fmt.Errorf("error creating directory %s: %v", filepath.Dir(v), err)
	// 		}
	// 		tmpDir, err := os.MkdirTemp(opts.Workdir, "*")
	// 		if err != nil {
	// 			return fmt.Errorf("error creating temporary directory: %v", err)
	// 		}
	// 		if err := os.Symlink(tmpDir, v); err != nil {
	// 			return fmt.Errorf("error linking volume %s: %v", v, err)
	// 		}
	// 	}
	// }

	if err := archiver.Untar(tmpFS, opts.RootFS, opts.Exclusions); err != nil {
		return err
	}

	env := cfg.Config.Env
	if opts.Eval {
		fmt.Println(strings.Join(env, "\n"))
	}

	logrus.Infof("Image %s has been loaded!\n", opts.Image)
	return nil
}

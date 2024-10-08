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

func load(conf *CommandOptions) error {

	img, err := save(conf)
	if err != nil {
		return err
	}

	tmpFS := filepath.Join(conf.Workdir, ".fs_"+util.Slugify(conf.Image)+".tar")
	defer os.Remove(tmpFS)

	if err := img.Export(tmpFS); err != nil {
		return err
	}

	logrus.Debugf("Fetching config file for image %s", img.Name)
	cfg, err := img.Config()
	if err != nil {
		return fmt.Errorf("error getting config from image %s: %v", img, err)
	}

	if err := cleanup(conf); err != nil {
		return err
	}

	// if conf.Volumes {
	// 	for v := range cfg.Config.Volumes {
	// 		if err := os.MkdirAll(filepath.Dir(v), os.ModePerm); err != nil {
	// 			return fmt.Errorf("error creating directory %s: %v", filepath.Dir(v), err)
	// 		}
	// 		tmpDir, err := os.MkdirTemp(conf.Workdir, "*")
	// 		if err != nil {
	// 			return fmt.Errorf("error creating temporary directory: %v", err)
	// 		}
	// 		if err := os.Symlink(tmpDir, v); err != nil {
	// 			return fmt.Errorf("error linking volume %s: %v", v, err)
	// 		}
	// 	}
	// }

	if err := archiver.Untar(tmpFS, conf.RootFS, conf.Exclusions); err != nil {
		return err
	}

	env := cfg.Config.Env
	if conf.Eval {
		fmt.Println(strings.Join(env, "\n"))
	}

	logrus.Infof("Image %s has been loaded!\n", conf.Image)
	return nil
}

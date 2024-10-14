package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/kukaryambik/givme/pkg/listpaths"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func proot(opts *CommandOptions) error {

	imgSlug := util.Slugify(opts.Image)
	opts.RootFS = filepath.Join(opts.Workdir, imgSlug, "rootfs")
	newExclusions, err := listpaths.Excl("/", append(opts.Exclusions, "!"+opts.RootFS))
	if err != nil {
		return err
	}
	opts.Exclusions = newExclusions

	img, err := load(opts)
	if err != nil {
		return err
	}

	imgConf, err := img.Config()
	if err != nil {
		return err
	}
	cfg := imgConf.Config

	bin := filepath.Join(util.GetExecDir(), "proot")
	cmd := exec.Command(bin)

	// add rootfs
	cmd.Args = append(cmd.Args, fmt.Sprintf("--rootfs=%s", opts.RootFS))

	// add user
	expr, err := regexp.Compile("^[0-9]+(:[0-9]+)?$")
	if err != nil {
		return fmt.Errorf("error compiling regex: %v", err)
	}
	if expr.MatchString(cfg.User) {
		cmd.Args = append(cmd.Args, fmt.Sprintf("--change-id=%s", cfg.User))
	} else {
		cmd.Args = append(cmd.Args, "-0")
	}

	// add workdir
	if cfg.WorkingDir != "" {
		cmd.Args = append(cmd.Args, fmt.Sprintf("--cwd=%s", cfg.WorkingDir))
	} else {
		cmd.Args = append(cmd.Args, "--cwd=/")
	}

	// add mounts
	for _, e := range opts.Exclusions {
		f := fmt.Sprintf("--bind=%s", e)
		cmd.Args = append(cmd.Args, f)
	}

	// add volumes
	for v := range cfg.Volumes {
		dir := filepath.Join(opts.Workdir, imgSlug, v)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("error creating directory %s: %v", dir, err)
		}
		f := fmt.Sprintf("--bind=%s", dir+":"+v)
		cmd.Args = append(cmd.Args, f)
	}

	// add shell
	if cfg.Shell != nil {
		cmd.Args = append(cmd.Args, cfg.Shell...)
	}

	// add entrypoint
	if opts.Entrypoint != "" {
		logrus.Debugln("Entrypoint:", opts.Entrypoint)
		cfg.Entrypoint = []string{opts.Entrypoint}
	}
	if cfg.Entrypoint != nil {
		cmd.Args = append(cmd.Args, cfg.Entrypoint...)
	}

	// add cmd
	if opts.Cmd != nil {
		logrus.Debugln("Cmd:", opts.Cmd)
		cfg.Cmd = opts.Cmd
	}
	if cfg.Cmd != nil {
		cmd.Args = append(cmd.Args, cfg.Cmd...)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = cfg.Env

	logrus.Debugln(cmd.Args)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running proot: %v", err)
	}

	return nil
}

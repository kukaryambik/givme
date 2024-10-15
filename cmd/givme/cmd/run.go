package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

func run(opts *CommandOptions) error {

	// Get the image
	img, err := save(opts)
	if err != nil {
		return err
	}

	// Create the image workspace
	dir, err := image.MkImageDir(opts.Workdir, opts.Image)
	if err != nil {
		return err
	}

	// Get the image config
	imgConf, err := img.Config()
	if err != nil {
		return err
	}
	cfg := imgConf.Config

	// Configure ignored paths
	ignoreConf := paths.Ignore(opts.IgnorePaths).ExclFromList(opts.RootFS)
	ignores, err := ignoreConf.AddPaths(opts.Workdir).List()
	if err != nil {
		return err
	}

	bin := filepath.Join(paths.GetExecDir(), "proot")
	cmd := exec.Command(bin, "--kill-on-exit")

	// add extra flags
	cmd.Args = append(cmd.Args, opts.ProotFlags...)

	// add rootfs
	cmd.Args = append(cmd.Args, fmt.Sprintf("--rootfs=%s", opts.RootFS))

	// add user
	expr, err := regexp.Compile("^[0-9]+(:[0-9]+)?$")
	if err != nil {
		return fmt.Errorf("error compiling regex: %v", err)
	}
	if opts.ProotUser != "" {
		cfg.User = opts.ProotUser
	}
	if expr.MatchString(cfg.User) {
		logrus.Debugf("User %s is numeric", cfg.User)
		cmd.Args = append(cmd.Args, fmt.Sprintf("--change-id=%s", cfg.User))
	} else {
		logrus.Debugf("User %s is not numeric", cfg.User)
		cmd.Args = append(cmd.Args, "-0")
	}

	// add workdir
	if opts.ProotCwd != "" {
		cfg.WorkingDir = opts.ProotCwd
	}
	if cfg.WorkingDir != "" {
		cmd.Args = append(cmd.Args, fmt.Sprintf("--cwd=%s", cfg.WorkingDir))
	} else {
		cmd.Args = append(cmd.Args, "--cwd=/")
	}

	// add mounts
	for _, e := range ignores {
		f := fmt.Sprintf("--bind=%s", e)
		cmd.Args = append(cmd.Args, f)
	}

	for _, m := range opts.ProotMounts {
		f := fmt.Sprintf("--bind=%s", m)
		cmd.Args = append(cmd.Args, f)
	}

	// add volumes
	for v := range cfg.Volumes {
		tmpDir := filepath.Join(dir, "vol_"+util.Slugify(v))
		if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
			return fmt.Errorf("error creating directory %s: %v", dir, err)
		}
		f := fmt.Sprintf("--bind=%s", tmpDir+":"+v)
		cmd.Args = append(cmd.Args, f)
	}

	// add shell
	if cfg.Shell != nil {
		cmd.Args = append(cmd.Args, cfg.Shell...)
	}

	// add entrypoint
	if opts.ProotEntrypoint != "" {
		logrus.Debugln("Entrypoint:", opts.ProotEntrypoint)
		cfg.Entrypoint = []string{opts.ProotEntrypoint}
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

	// Export the image filesystem to the tar file
	tmpFS := filepath.Join(dir, "fs.tar")
	defer os.Remove(tmpFS)
	if err := img.Export(tmpFS); err != nil {
		return err
	}

	// Clean up the rootfs
	if opts.Cleanup {
		if err := cleanup(opts); err != nil {
			return err
		}
	}

	// Untar the filesystem
	if err := archiver.Untar(tmpFS, opts.RootFS, ignores); err != nil {
		return err
	}

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running proot: %v", err)
	}

	return nil
}

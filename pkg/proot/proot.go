package proot

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"regexp"

	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

const (
	// Default binary path
	DefaultBinPath = "proot"
)

type ProotConf struct {
	// Basic configuration
	BinPath string   // Path to proot binary
	Command []string // List of commands
	Env     []string // Environment variables

	// Flags
	ChangeID      string   `flag:"change-id"`    // Make current user and group appear as *string* "uid:gid".
	KillOnExit    bool     `flag:"kill-on-exit"` // Kill all processes on command exit.
	Link2Symlink  bool     `flag:"link2symlink"` // Enable the link2symlink extension.
	MixedMode     bool     // Disable the mixed-execution feature.
	Binds         []string `flag:"bind"`           // Make the content of *path* accessible in the guest rootfs.
	Netcoop       bool     `flag:"netcoop"`        // Enable the network cooperation mode.
	Ports         []string `flag:"port"`           // Map ports to others with the syntax as *string* "port_in:port_out".
	RootFS        string   `flag:"rootfs"`         // Use *path* as the new guest root file-system, default is /.
	Verbose       int      `flag:"verbose"`        // Set the level of debug information to *value*.
	Workdir       string   `flag:"cwd"`            // Set the initial working directory to *path*.
	KernelRelease string   `flag:"kernel-release"` // Make current kernel appear as kernel release *string*.

	// Environment variables
	LibraryPath           string `env:"LD_LIBRARY_PATH"`
	DontPolluteRootfs     bool   `env:"PROOT_DONT_POLLUTE_ROOTFS"`
	ForceForeignBinary    bool   `env:"PROOT_FORCE_FOREIGN_BINARY"`
	ForceKompat           bool   `env:"PROOT_FORCE_KOMPAT"`
	IgnoreMissingBindings bool   `env:"PROOT_IGNORE_MISSING_BINDINGS"`
	Loader                string `env:"PROOT_LOADER"`
	Loader32              string `env:"PROOT_LOADER_32"`
	NoSeccomp             bool   `env:"PROOT_NO_SECCOMP"`
	TmpDir                string `env:"PROOT_TMP_DIR"`

	ExtraFlags []string // Extra flags to pass to proot
}

func hasTag(s *reflect.StructTag, t string) bool {
	_, ok := s.Lookup(t)
	return ok
}

func (cfg *ProotConf) Cmd() *exec.Cmd {
	// Create the proot command
	cmd := exec.Command(util.Coalesce(cfg.BinPath, DefaultBinPath))
	logrus.Debugf("Creating proot command: %s", cmd.Path)

	cmd.Env = append(cmd.Env, cfg.Env...)

	args := util.CleanList(cfg.ExtraFlags)

	// Add mixed mode
	args = append(args, fmt.Sprintf("--mixed-mode %v", cfg.MixedMode))

	// check UID:GID
	expr := regexp.MustCompile(`^[0-9]+(:[0-9]+)?$`)
	if !expr.MatchString(cfg.ChangeID) {
		logrus.Warnf("UID:GID %s is not numeric", cfg.ChangeID)
		cfg.ChangeID = ""
	}

	val := reflect.ValueOf(*cfg)
	t := reflect.TypeOf(*cfg)

	for i := 0; i < val.NumField(); i++ {
		v := val.Field(i)
		if !v.IsValid() || v.IsZero() {
			continue
		}

		tag := t.Field(i).Tag
		switch {
		case hasTag(&tag, "flag"):
			flag := tag.Get("flag")

			switch v.Kind() {
			case reflect.Bool:
				args = append(args, "--"+flag)
				logrus.Tracef("Added flag: --%s", flag)
			case reflect.Slice:
				for i := 0; i < v.Len(); i++ {
					elem := v.Index(i).Interface()
					args = append(args, fmt.Sprintf("--%s=%v", flag, elem))
					logrus.Tracef("Added flag: --%s=%v", flag, elem)
				}
			default:
				args = append(args, fmt.Sprintf("--%s=%v", flag, v))
				logrus.Tracef("Added flag: --%s=%v", flag, v)
			}

		case hasTag(&tag, "env"):
			env := tag.Get("env")

			switch v.Kind() {
			case reflect.Bool:
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=1", env))
				logrus.Debugf("Set environment variable: %s=1", env)
			default:
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%v", env, v))
				logrus.Debugf("Set environment variable: %s=%v", env, v)
			}
		}
	}

	// Add command
	if len(cfg.Command) > 0 {
		args = append(args, cfg.Command...)
		logrus.Debugf("Added command: %v (%v)", cfg.Command, len(cfg.Command))
	}

	// Remove empty strings
	for _, str := range args {
		if str != "" {
			cmd.Args = append(cmd.Args, str)
		}
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Log final command and environment variables
	logrus.Debugf("Final command: %v", args)
	logrus.Debugf("Environment variables: %v", cmd.Env)

	return cmd
}

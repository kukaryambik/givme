package cmd

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
)

// PrepareEntrypoint prepares the command to run in the container.
// If opts.Entrypoint is provided, it overrides the entrypoint from the image.
// If the image has no entrypoint, it defaults to /bin/sh.
// If the image has no command, it defaults to the command provided.
func (opts *CommandOptions) PrepareEntrypoint(cfg *v1.Config) []string {
	shell := util.Coalesce(util.CleanList(cfg.Shell), []string{"/bin/sh", "-c"})

	var cmd []string
	if len(opts.Entrypoint) > 0 {
		// If Entrypoint is provided, use the last element as the executable and append the remaining arguments.
		cmd = append(opts.Entrypoint[len(opts.Entrypoint)-1:], opts.Cmd...)
	} else {
		// If Entrypoint is not provided, use the image's entrypoint and command.
		cmd = append(
			util.CleanList(cfg.Entrypoint),
			util.Coalesce(util.CleanList(opts.Cmd), util.CleanList(cfg.Cmd))...,
		)
	}

	entrypoint := util.Coalesce(util.CleanList(cmd), shell[:1])
	logrus.Debugf("Entrypoint: %v", entrypoint)
	return entrypoint
}

// PrepareEnvForEval prepares the environment variables for the eval command
// If opts.OverwriteEnv is true, it overwrites the existing environment variables.
func (opts *CommandOptions) PrepareEnvForEval(cfg *v1.Config, saveToFile bool) (string, error) {
	currentEnv := envars.ToMap(os.Environ())
	imgEnv := envars.ToMap(cfg.Env)
	oldEnv, err := envars.FromFile(imgEnv, defaultDotEnvFile(), saveToFile)
	if err != nil {
		return "", err
	}

	unsetEnv := envars.Uniq(true, oldEnv, currentEnv)
	setEnv := envars.Merge(make(map[string]string), imgEnv)
	if !opts.OverwriteEnv {
		setEnv = envars.UniqKeys(imgEnv, envars.Uniq(false, currentEnv, oldEnv))
	}
	setEnv["PATH"] = strings.Trim(imgEnv["PATH"]+":"+util.GetExecDir(), ": ")

	unsetStr := ""
	if len(unsetEnv) > 0 {
		unsetStr = fmt.Sprintf("unset %s;\n", strings.Join(envars.ToSlice(true, unsetEnv), " "))
	}
	setStr := ""
	if len(setEnv) > 0 {
		setStr = fmt.Sprintf("export %s;\n", strings.Join(envars.ToSlice(true, setEnv), " "))
	}

	env := strings.TrimSpace(unsetStr + setStr)
	logrus.Debugf("Environment variables for eval:\n%s", env)
	return env, nil
}

// PrepareEnvForExec prepares the environment variables for the exec command
// It takes the current environment variables, the environment variables from the image,
// and the saved environment variables from the previous image, and returns a slice of strings
// that can be used as environment variables for the exec command.
func (opts *CommandOptions) PrepareEnvForExec(cfg *v1.Config) ([]string, error) {
	// Get the current environment variables
	currentEnv := envars.ToMap(os.Environ())
	// Get the environment variables from the image
	imageEnv := envars.ToMap(cfg.Env)

	// Check if the parent process is the same as the current process
	var save bool
	pname, _ := util.GetParentProcessName()
	logrus.Debugf("Parent process name: %s", pname)
	if pname == "" {
		save = true
	}

	// Get the environment variables saved in the file
	savedEnv, err := envars.FromFile(imageEnv, defaultDotEnvFile(), save)
	if err != nil {
		return nil, err
	}

	// Calculate the difference between the current environment and the saved environment
	// This is used to determine which environment variables to set.
	diff := envars.Uniq(false, currentEnv, savedEnv)

	// If opts.OverwriteEnv is true, use the environment variables from the image
	// Otherwise, use the environment variables from the saved environment.
	env := envars.Merge(imageEnv, diff)
	if opts.OverwriteEnv {
		env = envars.Merge(diff, imageEnv)
	}

	// Set the PATH environment variable to include the path to the current executable
	env["PATH"] = strings.Trim(imageEnv["PATH"]+":"+util.GetExecDir(), ": ")
	syscall.Setenv("PATH", env["PATH"])

	// Format the environment variables as a slice of strings
	envSlice := envars.ToSlice(false, env)
	logrus.Debugf("Environment variables for exec: %q", envSlice)
	return envSlice, nil
}

package cmd

import (
	"fmt"
	"os"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/kukaryambik/givme/pkg/envars"
	"github.com/kukaryambik/givme/pkg/util"
)

func (opts *CommandOptions) PrepareEntrypoint(cfg *v1.Config) []string {
	shell := util.Coalesce(util.CleanList(cfg.Shell), []string{"/bin/sh", "-c"})
	var args []string
	if len(opts.Entrypoint) > 0 {
		args = append(opts.Entrypoint[len(opts.Entrypoint)-1:], opts.Cmd...)
	} else {
		args = append(
			util.CleanList(cfg.Entrypoint),
			util.Coalesce(util.CleanList(opts.Cmd), util.CleanList(cfg.Cmd))...,
		)
	}
	args = append([]string{"exec"}, util.Coalesce(util.CleanList(args), shell[:1])...)
	return util.CleanList(append(shell, strings.Join(args, " ")))
}

// prepareEnvForEval prepares the environment variables for the eval command
func (opts *CommandOptions) PrepareEnvForEval(cfg *v1.Config, saveToFile bool) (string, error) {
	current := envars.ToMap(os.Environ())
	new := envars.ToMap(cfg.Env)
	old, err := envars.FromFile(new, defaultDotEnvFile(), saveToFile)
	if err != nil {
		return "", err
	}

	unset := envars.Uniq(true, old, current)
	set := envars.Merge(make(map[string]string), new)
	if !opts.OverwriteEnv {
		set = envars.UniqKeys(new, envars.Uniq(false, current, old))
	}
	set["PATH"] = strings.Trim(new["PATH"]+":"+util.GetExecDir(), ": ")

	var unsetStr, setStr string
	if len(unset) > 0 {
		unsetStr = "unset " + strings.Join(envars.ToSlice(true, unset), " ") + ";"
	}
	if len(set) > 0 {
		setStr = "export " + strings.Join(envars.ToSlice(true, set), " ") + ";"
	}

	return strings.TrimSpace(fmt.Sprintf("%s\n%s", unsetStr, setStr)), nil
}

// prepareEnvForExec prepares the environment variables for the exec command
func (opts *CommandOptions) PrepareEnvForExec(cfg *v1.Config) ([]string, error) {
	current := envars.ToMap(os.Environ())
	new := envars.ToMap(cfg.Env)
	old, err := envars.FromFile(new, defaultDotEnvFile(), false)
	if err != nil {
		return nil, err
	}

	diff := envars.Uniq(false, current, old)
	env := envars.Merge(new, diff)
	if opts.OverwriteEnv {
		env = envars.Merge(diff, new)
	}
	env["PATH"] = strings.Trim(new["PATH"]+":"+util.GetExecDir(), ": ")

	return envars.ToSlice(false, env), nil
}

package cmd

import (
	"fmt"

	"github.com/kukaryambik/givme/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func ApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apply [flags] IMAGE",
		Aliases: []string{"a", "an", "the"},
		Example: fmt.Sprintf("source <(%s apply alpine)", AppName),
		Short:   "Extract the image filesystem and print prepared environment variables to stdout",
		Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Image = args[0]
			cmd.SilenceUsage = true
			err := opts.Apply()
			if err != nil {
				fmt.Print("false")
			}
			return err
		},
	}

	cmd.Flags().BoolVar(
		&opts.Update, "update", opts.Update, "Update the image instead of using existing file")
	cmd.Flags().BoolVar(
		&opts.OverwriteEnv, "overwrite-env", opts.OverwriteEnv, "Overwrite current environment variables with new ones from the image")
	cmd.Flags().BoolVar(
		&opts.NoPurge, "no-purge", opts.NoPurge, "Do not purge the root directory before unpacking the image")

	return cmd
}

func (opts *CommandOptions) Apply() error {

	img, err := opts.Extract()
	if err != nil {
		return err
	}

	logrus.Debugf("Fetching config file for image %s", img.Name)
	cfg, err := img.Config()
	if err != nil {
		return fmt.Errorf("error getting config from image %s: %v", img, err)
	}

	logrus.Info("Preparing environment variables")

	outRedirected := util.IsOutRedirected()
	if !outRedirected {
		logrus.Warnf(
			"Output is not redirected!\n"+
				"It is strongly recommended to use this command in conjunction with source or eval. For example:\n"+
				"– `source <(%s apply %s)`\n"+
				"– `eval $(%s apply %s)`",
			AppName, opts.Image, AppName, opts.Image,
		)
		logrus.Info("Image environment variables will not be saved to a file")
	}

	// Prepare environment variables
	env, err := opts.PrepareEnvForEval(&cfg.Config, outRedirected)
	if err != nil {
		return err
	}

	fmt.Println(env)

	return nil
}

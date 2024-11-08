package cmd

import (
	"github.com/kukaryambik/givme/pkg/image"
	"github.com/kukaryambik/givme/pkg/paths"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func extractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "extract [flags] IMAGE",
		Aliases: []string{"ex", "ext", "unpack"},
		Short:   "Extract the image filesystem",
		Args:    cobra.ExactArgs(1), // Ensure exactly 1 argument is provided
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Image = args[0]
			cmd.SilenceUsage = true
			_, err := opts.Extract()
			return err
		},
	}

	cmd.Flags().BoolVar(
		&opts.Update, "update", opts.Update, "Update the image instead of using existing file")

	return cmd
}

// Extract extracts the image filesystem to opts.RootFS, using the same ignores
// as Save. If opts.NoPurge is false, it also purges the rootfs before extraction.
// It returns the extracted image.
func (opts *CommandOptions) Extract() (*image.Image, error) {

	// Get an image
	img, err := opts.Save()
	if err != nil {
		return nil, err
	}

	// Configure ignored paths
	ignoreConf := paths.Ignore(opts.IgnorePaths).ExclFromList(opts.RootFS)
	ignores, err := ignoreConf.AddPaths(opts.Workdir).List()
	if err != nil {
		return nil, err
	}

	// Clean up the rootfs
	if !opts.NoPurge {
		logrus.Infof("Purging rootfs '%s'", opts.RootFS)
		if err := paths.Rmrf(opts.RootFS, ignores); err != nil {
			return nil, err
		}
	}

	// Untar the filesystem
	if err := image.Extract(img, opts.RootFS, ignores...); err != nil {
		return nil, err
	}

	return img, nil
}

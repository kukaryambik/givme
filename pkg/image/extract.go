package image

import (
	"io"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/kukaryambik/givme/pkg/archiver"
	"github.com/sirupsen/logrus"
)

func Extract(img *Image, rootfs string, ignore ...string) error {

	logrus.Infof("Extracting filesystem to %q", rootfs)

	// Untar the filesystem
	reader, writer := io.Pipe()
	go func() {
		if err := crane.Export(img.Image, writer); err != nil {
			writer.CloseWithError(err)
			return
		}
		writer.Close()
	}()

	if err := archiver.Untar(reader, rootfs, ignore); err != nil {
		return err
	}

	return nil
}

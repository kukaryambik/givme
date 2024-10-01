package main

import (
	"os"

	"github.com/kukaryambik/givme/cmd/givme/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

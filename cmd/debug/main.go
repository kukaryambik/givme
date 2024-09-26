package main

import (
	"fmt"
	"path/filepath"
)

func main() {
	a := []string{
		"/",
		"//",
		"/bin/bash",
		"./test",
	}

	for _, a := range a {
		a, _ = filepath.Abs(a)
		fmt.Println(a)
	}
}

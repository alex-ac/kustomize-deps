package main

import (
	"fmt"
	"os"

	kdeps "github.com/alex-ac/kustomize-deps"
)

func main() {
	if err := kdeps.MakeCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

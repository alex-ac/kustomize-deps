package kdeps

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"
)

type Arguments struct {
	Input  string
	Output string
	Target string
}

func MakeCommand(fsys fs.FS) (cmd *cobra.Command) {
	args := &Arguments{}
	cmd = &cobra.Command{
		Use:     "kustomize-deps -i dir -o deployment.stamp.d -t deployment.stamp",
		Short:   "Generate make-compatible depfile with all files used by kustomize deployment",
		PreRunE: func(*cobra.Command, []string) error { return ValidateArguments(args) },
		RunE:    func(*cobra.Command, []string) error { return Run(fsys, *args) },
	}
	cmd.Flags().StringVarP(&args.Input, "input", "i", "", "Path to the kustomization.")
	cmd.Flags().StringVarP(&args.Output, "output", "o", "", "Output file name.")
	cmd.Flags().StringVarP(&args.Target, "target", "t", "", "Makefile target file name.")

	return
}

func ValidateArguments(args *Arguments) error {
	if args.Input == "" {
		return fmt.Errorf("-i is a required argument")
	}

	if args.Output == "" {
		return fmt.Errorf("-o is a required argument")
	}

	if args.Target == "" {
		return fmt.Errorf("-t is a required argument")
	}

	return nil
}

func Run(fsys fs.FS, args Arguments) error {
	a := NewDepsAccumulator()
	if err := CollectKustomizationDeps(&a, fsys, args.Input); err != nil {
		return err
	}

	data := MarshalToDepFile(".", args.Target, a)
	oldData, err := os.ReadFile(args.Output)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
	default:
		return err
	}

	// Don't update file if nothing changed.
	if !bytes.Equal(data, oldData) {
		return os.WriteFile(args.Output, data, 0666)
	}

	return nil
}

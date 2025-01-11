package kdeps

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type Arguments struct {
	Input  string
	Output string
	Target string
}

const Docs = `Generate make-compatible depfile with all files used by kustomize deployment.

When using Makefile to orchestrate calls of deployment with kustomize, it's very
easy to call deployment when there are no changes in that deployment, because
kustomize does not report which files it uses during deployment.

You either need to manually specify the whole list of files in the dependencies
or you will call deployment when it is not necessary.

Kustomize-deps is here to solve this problem. Use it to manage depfile for
kustomize target.

## Example

Let's say we've got a project layout like this:

  Makefile
	deployment/
		base/
			deployment.yaml
			ingress.yaml
			kustomization.yaml
			service.yaml
		local/
			config.env
			kustomization.yaml
			namespace.yaml
			secret.env
		dev/
			kustomization.yaml
			...
		...
	Dockerfile
	cmd/
		service/
			main.go

And we use a target like this in Makefile to perform a local deployment:

	.PHONY: deploy-local
	deploy-local: build/deploy-local/kustomization.yaml
		kustomize build deployment/local | kubectl apply -f - --server-side --prune -l app.kubernetes.io/managed-by=kustomize-local

	build/deploy-local/kustomization.yaml: build/service.imageid
		! [ -e $@ ] || rm -f $@
		mkdir -p $(dir $@)
		( cd $(dir $@) && \
			kustomize create --resources ../../deployment/local && \
			kustomize edit set image service=service@$$(<$<) )

	build/service.imageid: Dockerfile cmd/service/main.go
		docker build -t service:dev . --iidfile=$@

With kustomize-deps we can track dependencies precisely, and only run kustomize
if there are any changes:

	.PHONY: deploy-local
	deploy-local: build/deploy-local.stamp

	build/deploy-local.stamp: build/deploy-local/kustomization.yaml
		kustomize build deployment/local | kubectl apply -f - --server-side --prune -l app.kubernetes.io/managed-by=kustomize-local
		go run github.com/alex-ac/kustomize-deps/cmd/kustomize-deps -o $@.d -i $(dir $<) -t $@
	
	-include build/deploy-local.stamp.d

	build/deploy-local/kustomization.yaml: build/service.imageid
		! [ -e $@ ] || rm -f $@
		mkdir -p $(dir $@)
		( IMAGEID=$$(<$<) ; \
			cd $(dir $@) && \
			kustomize create --resources ../../deployment/local && \
			kustomize edit set image service=service@$$IMAGEID )

	build/service.imageid: Dockerfile cmd/service/main.go
		docker build -t service:dev . --iidfile=$@

What is this magic?

kustomize-deps reads kustomization.yaml in the same way as kustomize, but instead
of generating resource manifests, it simply collects paths to the inputs. And then
generates a depfile like this:

  build/deploy-local.stamp: build/deploy-local/kustomization.yaml deployment/local/kustomization.yaml deployment/local/config.env deployment/local/namespace.yaml deployment/local/secret.env deployment/base/kustomization.yaml deployment/base/ingress.yaml deployment/base/service.yaml deployment/base/deployment.yaml
  build/deploy-local.stamp.d: build/deploy-local/kustomization.yaml deployment/local/kustomization.yaml deployment/local/config.env deployment/local/namespace.yaml deployment/local/secret.env deployment/base/kustomization.yaml deployment/base/ingress.yaml deployment/base/service.yaml deployment/base/deployment.yaml

Which is included into your makefile:

  -include build/deploy-local.stamp.d

That lets makefile now the whole list of deployment dependencies.

Why there are two calls of kustomize-deps? Because if you use kustomize with
helm charts, kustomize can download those charts and place them as files.
It's impossible to know whole list of files in the chart before the download
so second run picks up those files and adds them to the dependency list.
On the second run it is recommended to use --keep-mtime flag to prevent
dependency cycle.
`

func MakeCommand() (cmd *cobra.Command) {
	args := &Arguments{}
	cmd = &cobra.Command{
		Use:     "kustomize-deps -i dir -o deployment.stamp.d -t deployment.stamp",
		Short:   "Generate make-compatible depfile with all files used by kustomize deployment",
		Long:    Docs,
		PreRunE: func(*cobra.Command, []string) error { return ValidateArguments(args) },
		RunE:    func(*cobra.Command, []string) error { return Run(*args) },
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

func Run(args Arguments) error {
	var err error
	args.Output, err = filepath.Abs(args.Output)
	if err != nil {
		return err
	}
	args.Input, err = filepath.Abs(args.Input)
	if err != nil {
		return err
	}

	a := NewDepsAccumulator()
	if err := CollectKustomizationDeps(&a, args.Input); err != nil {
		return err
	}

	data := MarshalToDepFile(args.Input, args.Target, a)
	var oldData []byte
	oldData, err = os.ReadFile(args.Output)
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

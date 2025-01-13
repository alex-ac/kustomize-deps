package kdeps

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"sigs.k8s.io/kustomize/api/types"
)

var RecognizedKustomizationFileName = []string{
	"kustomization.yaml",
	"kustomization.yml",
	"Kustomization",
}

func NewErrMissingKustomization(path string) error {
	text := bytes.NewBufferString("unable to find one of ")
	for i, kf := range RecognizedKustomizationFileName {
		switch i {
		case 0:
		case len(RecognizedKustomizationFileName) - 1:
			text.WriteString(", or ")
		default:
			text.WriteString(", ")
		}
		text.WriteRune('\'')
		text.WriteString(kf)
		text.WriteRune('\'')
	}
	text.WriteString(" in directory '")
	text.WriteString(path)
	text.WriteRune('\'')
	return errors.New(text.String())
}

func LoadKustFile(fsys fs.FS, path string) (content []byte, kustFileName string, err error) {
	match := 0
	for _, kf := range RecognizedKustomizationFileName {
		var c []byte
		kf := filepath.Join(path, kf)
		c, err = fs.ReadFile(fsys, kf)
		if c != nil {
			match += 1
			content = c
			kustFileName = kf
		}
	}

	switch match {
	case 0:
		err = NewErrMissingKustomization(path)
	case 1:
		err = nil
		break
	default:
		content = nil
		kustFileName = ""
		err = fmt.Errorf("Found multiple kustomization files under: %s", path)
	}

	return
}

func LoadKustTarget(fsys fs.FS, path string) (string, *types.Kustomization, error) {
	content, kustFileName, err := LoadKustFile(fsys, path)
	if err != nil {
		return "", nil, err
	}

	var k types.Kustomization
	if err := k.Unmarshal(content); err != nil {
		return "", nil, err
	}

	k.FixKustomization()
	return kustFileName, &k, nil
}

type ResourceType int

const (
	FileResource ResourceType = iota
	DirResource
	RemoteResource
)

func GetResourceType(fsys fs.FS, path string) (ResourceType, error) {
	switch info, err := fs.Stat(fsys, path); {
	case err == nil && info.IsDir():
		return DirResource, nil
	case err == nil && !info.IsDir():
		return FileResource, nil
	case errors.Is(err, os.ErrNotExist):
		return RemoteResource, nil
	default:
		return FileResource, err
	}
}

func CollectGeneratorDeps(a *DepsAccumulator, fsys fs.FS, root string, args types.GeneratorArgs) error {
	for _, path := range args.FileSources {
		resolvedPath := filepath.Clean(filepath.Join(root, path))
		resourceType, err := GetResourceType(fsys, resolvedPath)
		if err != nil {
			return err
		}

		if resourceType != RemoteResource {
			a.AddDep(resolvedPath)
		} else {
			a.AddNonFileDep(resolvedPath)
		}
	}

	for _, path := range args.EnvSources {
		resolvedPath := filepath.Clean(filepath.Join(root, path))
		resourceType, err := GetResourceType(fsys, resolvedPath)
		if err != nil {
			return err
		}

		if resourceType != RemoteResource {
			a.AddDep(resolvedPath)
		} else {
			a.AddNonFileDep(resolvedPath)
		}
	}
	return nil
}

func CollectChart(a *DepsAccumulator, helmHome, name string) error {
	chartPath := filepath.Join(helmHome, name)
	return filepath.Walk(chartPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			a.AddDep(path)
		}
		return nil
	})
}

func CollectKustomizationDeps(a *DepsAccumulator, fsys fs.FS, root string) error {
	path, k, err := LoadKustTarget(fsys, root)
	if err != nil {
		return err
	}
	a.AddDep(path)

	if openAPIPath, exists := k.OpenAPI["path"]; exists {
		openAPIPath = filepath.Clean(filepath.Join(root, path))
		a.AddDep(openAPIPath)
	}

	for _, path := range k.Resources {
		resolvedPath := filepath.Clean(filepath.Join(root, path))
		resourceType, err := GetResourceType(fsys, resolvedPath)
		if err != nil {
			return err
		}
		switch resourceType {
		case FileResource:
			a.AddDep(resolvedPath)
		case DirResource:
			if err := CollectKustomizationDeps(a, fsys, resolvedPath); err != nil {
				return err
			}
		case RemoteResource:
			a.AddNonFileDep(path)
		}
	}

	for _, path := range k.Configurations {
		a.AddDep(filepath.Clean(filepath.Join(root, path)))
	}

	for _, path := range k.Crds {
		resolvedPath := filepath.Clean(filepath.Join(root, path))
		resourceType, err := GetResourceType(fsys, resolvedPath)
		if err != nil {
			return err
		}

		if resourceType != RemoteResource {
			a.AddDep(resolvedPath)
		} else {
			a.AddNonFileDep(resolvedPath)
		}
	}

	for _, configMapArgs := range k.ConfigMapGenerator {
		if err := CollectGeneratorDeps(a, fsys, root, configMapArgs.GeneratorArgs); err != nil {
			return err
		}
	}

	for _, secretArgs := range k.SecretGenerator {
		if err := CollectGeneratorDeps(a, fsys, root, secretArgs.GeneratorArgs); err != nil {
			return err
		}
	}

	for _, patch := range k.Patches {
		if patch.Path != "" {
			resolvedPath := filepath.Clean(filepath.Join(root, patch.Path))
			a.AddDep(resolvedPath)
		}
	}

	helmHome := filepath.Join(root, "charts")
	if helmGlobals := k.HelmGlobals; helmGlobals != nil {
		if helmGlobals.ChartHome != "" {
			helmHome = filepath.Join(root, helmGlobals.ChartHome)
		}
	}

	for _, chart := range k.HelmCharts {
		if err := CollectChart(a, helmHome, chart.Name); err != nil {
			return err
		}

		for _, filePath := range chart.AdditionalValuesFiles {
			resolvedPath := filepath.Join(root, filePath)
			a.AddDep(resolvedPath)
		}

		if chart.ValuesFile != "" {
			resolvedPath := filepath.Join(root, chart.ValuesFile)
			a.AddDep(resolvedPath)
		}
	}

	return nil
}

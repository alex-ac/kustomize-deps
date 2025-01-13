package kdeps

import (
	"bytes"
	"path/filepath"
	"sort"
)

type DepsAccumulator struct {
	Deps        map[string]struct{}
	NonFileDeps map[string]struct{}
}

func NewDepsAccumulator() DepsAccumulator {
	return DepsAccumulator{
		Deps:        map[string]struct{}{},
		NonFileDeps: map[string]struct{}{},
	}
}

func (a *DepsAccumulator) AddDep(file string) {
	a.Deps[file] = struct{}{}
}

func (a DepsAccumulator) HasDep(file string) bool {
	_, ok := a.Deps[file]
	return ok
}

func (a *DepsAccumulator) AddNonFileDep(url string) {
	a.NonFileDeps[url] = struct{}{}
}

func (a DepsAccumulator) HasNonFileDep(url string) bool {
	_, ok := a.NonFileDeps[url]
	return ok
}

func MarshalToDepFile(base, target string, a DepsAccumulator) []byte {
	text := bytes.NewBufferString(target)
	text.WriteRune(':')
	deps := make([]string, 0, len(a.Deps))
	for dep, _ := range a.Deps {
		path, err := filepath.Rel(base, dep)
		if err != nil {
			path = dep
		}
		deps = append(deps, path)
	}
	sort.Strings(deps)
	for _, dep := range deps {
		text.WriteRune(' ')
		text.WriteString(dep)
	}
	text.WriteRune('\n')

	if len(a.NonFileDeps) > 0 {
		text.WriteString(target)
		text.WriteString(": X_KUSTOMIZE_NON_FILE_DEPS='")

		deps = make([]string, 0, len(a.NonFileDeps))
		for dep, _ := range a.NonFileDeps {
			deps = append(deps, dep)
		}
		sort.Strings(deps)

		for i, dep := range deps {
			if i != 0 {
				text.WriteRune(' ')
			}
			text.WriteString(dep)
		}
		text.WriteString("'\n")
	}

	return text.Bytes()
}

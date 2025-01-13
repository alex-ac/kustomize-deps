package kdeps_test

import (
	"testing"

	. "github.com/alex-ac/kustomize-deps"
	"github.com/stretchr/testify/assert"
)

func TestAccumulator(t *testing.T) {
	assert := assert.New(t)

	a := NewDepsAccumulator()
	a.AddDep("/a/b.c")
	a.AddDep("/a/b.c")
	a.AddDep("/a/b/c.d")

	d := MarshalToDepFile("/a/", "b", a)
	assert.Equal("b: b.c b/c.d\n", string(d))

	a.AddNonFileDep("d.e")
	a.AddNonFileDep("d.e")
	a.AddNonFileDep("d.e.f")
	d = MarshalToDepFile("/a/", "b", a)
	assert.Equal("b: b.c b/c.d\nb: X_KUSTOMIZE_NON_FILE_DEPS='d.e d.e.f'\n", string(d))
}

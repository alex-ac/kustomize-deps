package kdeps_test

import (
	"embed"
	"io/fs"
	"testing"

	. "github.com/alex-ac/kustomize-deps"
	"github.com/stretchr/testify/assert"
)

//go:embed test/fixture
var fixtureFS embed.FS
var fixture = mustSub(fixtureFS, "test/fixture")

func mustSub(fsys fs.FS, path string) fs.FS {
	var err error
	fsys, err = fs.Sub(fsys, path)
	if err != nil {
		panic(err)
	}
	return fsys
}

func TestCollectors(t *testing.T) {
	assert := assert.New(t)

	a := NewDepsAccumulator()
	err := CollectKustomizationDeps(&a, fixture, "local-gen")
	assert.NoError(err)

	assert.True(a.HasDep("local-gen/kustomization.yaml"))
	assert.True(a.HasDep("local/kustomization.yaml"))
	assert.True(a.HasDep("local/deployment.yaml"))
	assert.True(a.HasDep("local/config.env"))
	assert.True(a.HasDep("local/secret.env"))
	assert.True(a.HasDep("local/tls.crt"))
	assert.True(a.HasDep("local/tls.key"))
	assert.True(a.HasDep("base/kustomization.yaml"))
	assert.True(a.HasDep("base/deployment.yaml"))
	assert.True(a.HasDep("base/service.yaml"))
	assert.True(a.HasDep("base/ingress.yaml"))
	assert.True(a.HasDep("base/config.ini"))
	assert.False(a.HasDep("dev/kustomization.yaml"))
}

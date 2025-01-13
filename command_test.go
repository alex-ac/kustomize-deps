package kdeps_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/alex-ac/kustomize-deps"
	"github.com/stretchr/testify/assert"
)

func TestCommand(t *testing.T) {
	assert := assert.New(t)
	tempdir := t.TempDir()
	output := filepath.Join(tempdir, "local-gen.stamp.d")

	expectedOut := strings.Join([]string{
		"local-gen.stamp:",
		"base/config.ini",
		"base/deployment.yaml",
		"base/ingress.yaml",
		"base/kustomization.yaml",
		"base/service.yaml",
		"local-gen/kustomization.yaml",
		"local/config.env",
		"local/deployment.yaml",
		"local/kustomization.yaml",
		"local/secret.env",
		"local/tls.crt",
		"local/tls.key",
	}, " ") + "\n"

	cmd := MakeCommand(fixture)
	cmd.SetArgs([]string{"-i", "local-gen", "-t", "local-gen.stamp", "-o", output})
	assert.NoError(cmd.Execute())
	out, err := os.ReadFile(output)
	assert.NoError(err)
	assert.Equal(expectedOut, string(out))
}

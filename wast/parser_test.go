package wast

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	specDir := filepath.Join("..", "internal", "testdata", "spec")

	entries, err := ioutil.ReadDir(specDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".wast" {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			f, err := os.Open(filepath.Join(specDir, entry.Name()))
			require.NoError(t, err)
			defer f.Close()

			_, err = ParseScript(NewScanner(f))
			assert.NoError(t, err)
		})
	}
}

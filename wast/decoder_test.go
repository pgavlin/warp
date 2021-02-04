package wast

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecoder(t *testing.T) {
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

			s, err := ParseScript(NewScanner(f))
			require.NoError(t, err)

			for _, cmd := range s.Commands {
				module, ok := cmd.(*Module)
				if !ok {
					continue
				}

				_, err = module.Decode()
				assert.NoError(t, err)
			}
		})
	}
}

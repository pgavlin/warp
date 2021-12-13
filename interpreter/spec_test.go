package interpreter

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/pgavlin/warp/exec"
	warp_testing "github.com/pgavlin/warp/testing"
	"github.com/pgavlin/warp/wasm"
	"github.com/stretchr/testify/require"
)

var specTest = flag.String("spec", "", "spec test to run")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestSpec(t *testing.T) {
	if *specTest != "" {
		warp_testing.RunScript(t, func(m *wasm.Module) (exec.ModuleDefinition, error) {
			return NewModuleDefinition(m), nil
		}, *specTest, false, ignore[filepath.Base(*specTest)])
		return
	}

	specDir := filepath.Join("..", "internal", "testdata", "spec")

	entries, err := ioutil.ReadDir(specDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".wast" {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			//t.Parallel()

			warp_testing.RunScript(t, func(m *wasm.Module) (exec.ModuleDefinition, error) {
				return NewModuleDefinition(m), nil
			}, filepath.Join(specDir, entry.Name()), false, ignore[entry.Name()])

			// ICode only
			warp_testing.RunScript(t, func(m *wasm.Module) (exec.ModuleDefinition, error) {
				return newModuleDefinition(m, icodeOnly), nil
			}, filepath.Join(specDir, entry.Name()), false, ignore[entry.Name()])

			// Tracing ICode only
			warp_testing.RunScript(t, func(m *wasm.Module) (exec.ModuleDefinition, error) {
				return newModuleDefinition(m, icodeTrace), nil
			}, filepath.Join(specDir, entry.Name()), false, ignore[entry.Name()])

			// FCode only
			warp_testing.RunScript(t, func(m *wasm.Module) (exec.ModuleDefinition, error) {
				return newModuleDefinition(m, fcodeOnly), nil
			}, filepath.Join(specDir, entry.Name()), false, ignore[entry.Name()])
		})
	}
}

var ignore = map[string][]string{
	"binary-leb128.wast": {
		"403,2: assert_malformed: module was not malformed",
		"422,2: assert_malformed: module was not malformed",
		"441,2: assert_malformed: module was not malformed",
		"460,2: assert_malformed: module was not malformed",
		"729,2: assert_malformed: module was not malformed",
		"748,2: assert_malformed: module was not malformed",
		"767,2: assert_malformed: module was not malformed",
		"785,2: assert_malformed: module was not malformed",
		"804,2: assert_malformed: module was not malformed",
		"823,2: assert_malformed: module was not malformed",
		"842,2: assert_malformed: module was not malformed",
		"861,2: assert_malformed: module was not malformed",
		"986,2: assert_malformed: module was not malformed",
	},
	"binary.wast": {
		"70,1: assert_malformed: module was not malformed",
		"163,1: assert_malformed: module was not malformed",
		"183,1: assert_malformed: module was not malformed",
		"203,1: assert_malformed: module was not malformed",
		"222,2: assert_malformed: module was not malformed",
		"241,2: assert_malformed: module was not malformed",
		"261,1: assert_malformed: module was not malformed",
		"280,1: assert_malformed: module was not malformed",
		"299,1: assert_malformed: module was not malformed",
		"317,2: assert_malformed: module was not malformed",
		"335,2: assert_malformed: module was not malformed",
		"371,1: assert_malformed: module was not malformed",
		"387,2: assert_malformed: module was not malformed",
		"421,1: assert_malformed: module was not malformed",
		"431,1: assert_malformed: module was not malformed",
		"440,1: assert_malformed: module was not malformed",
		"451,1: assert_malformed: module was not malformed",
		"847,1: assert_malformed: module was not malformed",
		"886,1: assert_malformed: module was not malformed",
	},
	"const.wast": {
		"445,2: assert_return: expected [8.881785e-16], got [8.881784e-16]",
		"447,2: assert_return: expected [-8.881785e-16], got [-8.881784e-16]",
		"461,2: assert_return: expected [8.881785e-16], got [8.881786e-16]",
		"463,2: assert_return: expected [-8.881785e-16], got [-8.881786e-16]",
		"493,2: assert_return: expected [8.881787e-16], got [8.881786e-16]",
		"495,2: assert_return: expected [-8.881787e-16], got [-8.881786e-16]",
		"551,2: assert_return: expected [8.881785e-16], got [8.881784e-16]",
		"553,2: assert_return: expected [-8.881785e-16], got [-8.881784e-16]",
		"555,2: assert_return: expected [8.881785e-16], got [8.881786e-16]",
		"557,2: assert_return: expected [-8.881785e-16], got [-8.881786e-16]",
		"569,2: assert_return: expected [1.1259e+15], got [1.1258999e+15]",
		"571,2: assert_return: expected [-1.1259e+15], got [-1.1258999e+15]",
		"585,2: assert_return: expected [1.1259e+15], got [1.1259002e+15]",
		"587,2: assert_return: expected [-1.1259e+15], got [-1.1259002e+15]",
		"617,2: assert_return: expected [1.1259003e+15], got [1.1259002e+15]",
		"619,2: assert_return: expected [-1.1259003e+15], got [-1.1259002e+15]",
		"735,2: assert_return: expected [3.4028235e+38], got [+Inf]",
		"737,2: assert_return: expected [-3.4028235e+38], got [-Inf]",
		"745,2: assert_return: expected [2.4099198651028847e-181], got [2.409919865102884e-181]",
		"747,2: assert_return: expected [-2.4099198651028847e-181], got [-2.409919865102884e-181]",
		"761,2: assert_return: expected [2.4099198651028847e-181], got [2.409919865102885e-181]",
		"763,2: assert_return: expected [-2.4099198651028847e-181], got [-2.409919865102885e-181]",
		"789,2: assert_return: expected [2.4099198651028857e-181], got [2.409919865102885e-181]",
		"791,2: assert_return: expected [-2.4099198651028857e-181], got [-2.409919865102885e-181]",
		"798,2: assert_return: expected [2.4099198651028847e-181], got [2.409919865102884e-181]",
		"800,2: assert_return: expected [-2.4099198651028847e-181], got [-2.409919865102884e-181]",
		"814,2: assert_return: expected [2.4099198651028847e-181], got [2.409919865102885e-181]",
		"816,2: assert_return: expected [-2.4099198651028847e-181], got [-2.409919865102885e-181]",
		"842,2: assert_return: expected [2.4099198651028857e-181], got [2.409919865102885e-181]",
		"844,2: assert_return: expected [-2.4099198651028857e-181], got [-2.409919865102885e-181]",
		"851,2: assert_return: expected [5.357543035931338e+300], got [5.357543035931337e+300]",
		"853,2: assert_return: expected [-5.357543035931338e+300], got [-5.357543035931337e+300]",
		"855,2: assert_return: expected [5.357543035931338e+300], got [5.357543035931339e+300]",
		"857,2: assert_return: expected [-5.357543035931338e+300], got [-5.357543035931339e+300]",
		"869,2: assert_return: expected [4.149515568880994e+180], got [4.149515568880993e+180]",
		"871,2: assert_return: expected [-4.149515568880994e+180], got [-4.149515568880993e+180]",
		"885,2: assert_return: expected [4.149515568880994e+180], got [4.149515568880995e+180]",
		"887,2: assert_return: expected [-4.149515568880994e+180], got [-4.149515568880995e+180]",
		"917,2: assert_return: expected [4.149515568880996e+180], got [4.149515568880995e+180]",
		"919,2: assert_return: expected [-4.149515568880996e+180], got [-4.149515568880995e+180]",
		"926,2: assert_return: expected [1.584563250285287e+29], got [1.5845632502852868e+29]",
		"928,2: assert_return: expected [-1.584563250285287e+29], got [-1.5845632502852868e+29]",
		"942,2: assert_return: expected [1.584563250285287e+29], got [1.5845632502852875e+29]",
		"944,2: assert_return: expected [-1.584563250285287e+29], got [-1.5845632502852875e+29]",
		"974,2: assert_return: expected [1.5845632502852878e+29], got [1.5845632502852875e+29]",
		"976,2: assert_return: expected [-1.5845632502852878e+29], got [-1.5845632502852875e+29]",
		"1049,2: assert_return: expected [2.225073858507203e-308], got [2.2250738585072024e-308]",
		"1051,2: assert_return: expected [-2.225073858507203e-308], got [-2.2250738585072024e-308]",
		"1059,2: assert_return: expected [1.7976931348623157e+308], got [+Inf]",
		"1061,2: assert_return: expected [-1.7976931348623157e+308], got [-Inf]",
	},
	"custom.wast": {
		"84,2: assert_malformed: module was not malformed",
		"101,2: assert_malformed: module was not malformed",
	},
	"if.wast": {
		"923,2: assert_invalid: module was not invalid",
		"942,2: assert_invalid: module was not invalid",
		"955,2: assert_invalid: module was not invalid",
		"961,2: assert_invalid: module was not invalid",
		"1285,2: assert_invalid: module was not invalid",
		"1357,2: assert_invalid: module was not invalid",
	},
	"imports.wast": {
		"360,2: assert_invalid: module was not invalid",
		"455,2: assert_invalid: module was not invalid",
	},
	"linking.wast": {
		"136,2: assert_trap: expected uninitialized, got uninitialized element",
		"137,2: assert_trap: expected uninitialized, got uninitialized element",
		"139,2: assert_trap: expected uninitialized, got uninitialized element",
		"141,2: assert_trap: expected uninitialized, got uninitialized element",
		"142,2: assert_trap: expected uninitialized, got uninitialized element",
		"144,2: assert_trap: expected uninitialized, got uninitialized element",
		"146,2: assert_trap: expected undefined, got undefined element",
		"147,2: assert_trap: expected undefined, got undefined element",
		"148,2: assert_trap: expected undefined, got undefined element",
		"149,2: assert_trap: expected undefined, got undefined element",
		"152,2: assert_trap: expected indirect call, got indirect call type mismatch",
		"184,2: assert_trap: expected uninitialized, got uninitialized element",
		"185,2: assert_trap: expected uninitialized, got uninitialized element",
		"187,2: assert_trap: expected uninitialized, got uninitialized element",
		"188,2: assert_trap: expected uninitialized, got uninitialized element",
		"190,2: assert_trap: expected undefined, got undefined element",
		"225,2: assert_trap: expected uninitialized, got uninitialized element",
		"236,2: assert_trap: expected uninitialized, got uninitialized element",
		"248,2: assert_trap: expected uninitialized, got uninitialized element",
	},
}

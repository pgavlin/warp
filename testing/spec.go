package testing

import "github.com/pgavlin/warp/exec"

var SpecTest exec.ModuleDefinition = exec.NewHostModuleDefinition(func() (*specTest, error) {
	return &specTest{
		Global_i32: exec.NewGlobalI32(true, 666),
		Global_i64: exec.NewGlobalI64(true, 666),
		Global_f32: exec.NewGlobalF32(true, 0),
		Global_f64: exec.NewGlobalF64(true, 0),
		Table:      exec.NewTable(10, 20),
		Memory:     exec.NewMemory(1, 2),
	}, nil
})

type specTest struct {
	Global_i32 exec.Global
	Global_i64 exec.Global
	Global_f32 exec.Global
	Global_f64 exec.Global

	Table exec.Table

	Memory exec.Memory
}

func (st *specTest) Print() {
}

func (st *specTest) Print_i32(param int32) {
}

func (st *specTest) Print_i64(param int64) {
}

func (st *specTest) Print_f32(param float32) {
}

func (st *specTest) Print_f64(param float64) {
}

func (st *specTest) Print_i32_f32(param int32, param1 float32) {
}

func (st *specTest) Print_f64_f64(param, param1 float64) {
}

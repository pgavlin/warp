package wasi

type handle uint32

type pointer uint32

type list struct {
	pointer pointer
	length  int32
}

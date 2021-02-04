package exec

import "math"

func I32DivS(i1, i2 int32) int32 {
	if i1 == math.MinInt32 && i2 == -1 {
		panic(TrapIntegerOverflow)
	}
	return i1 / i2
}

func I64DivS(i1, i2 int64) int64 {
	if i1 == math.MinInt64 && i2 == -1 {
		panic(TrapIntegerOverflow)
	}
	return i1 / i2
}

func Fmax(z1, z2 float64) float64 {
	if math.IsNaN(z1) {
		return z1
	}
	if math.IsNaN(z2) {
		return z2
	}
	return math.Max(z1, z2)
}

func Fmin(z1, z2 float64) float64 {
	if math.IsNaN(z1) {
		return z1
	}
	if math.IsNaN(z2) {
		return z2
	}
	return math.Min(z1, z2)
}

func I32TruncS(z float64) int32 {
	z = math.Trunc(z)
	if math.IsNaN(z) {
		panic(TrapInvalidConversionToInteger)
	}
	if z < math.MinInt32 || z > math.MaxInt32 {
		panic(TrapIntegerOverflow)
	}
	return int32(z)
}

func I32TruncU(z float64) uint32 {
	z = math.Trunc(z)
	if math.IsNaN(z) {
		panic(TrapInvalidConversionToInteger)
	}
	if z <= -1 || z > math.MaxUint32 {
		panic(TrapIntegerOverflow)
	}
	return uint32(z)
}

func I64TruncS(z float64) int64 {
	z = math.Trunc(z)
	if math.IsNaN(z) {
		panic(TrapInvalidConversionToInteger)
	}
	if z < math.MinInt64 || z >= math.MaxInt64 {
		panic(TrapIntegerOverflow)
	}
	return int64(z)
}

func I64TruncU(z float64) uint64 {
	z = math.Trunc(z)
	if math.IsNaN(z) {
		panic(TrapInvalidConversionToInteger)
	}
	if z <= -1 || z >= math.MaxUint64 {
		panic(TrapIntegerOverflow)
	}
	return uint64(z)
}

func I32TruncSatS(z float64) int32 {
	switch {
	case math.IsNaN(z):
		return 0
	case math.IsInf(z, -1) || z <= math.MinInt32:
		return math.MinInt32
	case math.IsInf(z, 1) || z >= math.MaxInt32:
		return math.MaxInt32
	default:
		return int32(z)
	}
}

func I32TruncSatU(z float64) uint32 {
	switch {
	case math.IsNaN(z) || math.IsInf(z, -1) || z < 0:
		return 0
	case math.IsInf(z, 1) || z >= math.MaxUint32:
		return math.MaxUint32
	default:
		return uint32(z)
	}
}

func I64TruncSatS(z float64) int64 {
	switch {
	case math.IsNaN(z):
		return 0
	case math.IsInf(z, -1) || z <= math.MinInt64:
		return math.MinInt64
	case math.IsInf(z, 1) || z >= math.MaxInt64:
		return math.MaxInt64
	default:
		return int64(z)
	}
}

func I64TruncSatU(z float64) uint64 {
	switch {
	case math.IsNaN(z) || math.IsInf(z, -1) || z < 0:
		return 0
	case math.IsInf(z, 1) || z >= math.MaxUint64:
		return math.MaxUint64
	default:
		return uint64(z)
	}
}

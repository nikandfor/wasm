package wasm

import "math"

type (
	LowEncoder struct{}
)

func (e *LowEncoder) Int(b []byte, v int) []byte {
	return e.Uint64(b, uint64(v))
}

func (e *LowEncoder) Uint64(b []byte, v uint64) []byte {
	for {
		x := byte(v) & 0x7f
		v >>= 7

		if v != 0 {
			x |= 0x80
		}

		b = append(b, x)

		if x&0x80 == 0 {
			break
		}
	}

	return b
}

func (e *LowEncoder) Int64(b []byte, v int64) []byte {
	for {
		x := byte(v) & 0x7f
		s := byte(v) & 0x40
		v >>= 7

		if s == 0 && v != 0 || s != 0 && v != -1 {
			x |= 0x80
		}

		b = append(b, x)

		if x&0x80 == 0 {
			break
		}
	}

	return b
}

func (e *LowEncoder) Float64(b []byte, v float64) []byte {
	x := math.Float64bits(v)

	return append(b, byte(x), byte(x>>8), byte(x>>16), byte(x>>24), byte(x>>32), byte(x>>40), byte(x>>48), byte(x>>56))
}

func (e *LowEncoder) Name(b []byte, v string) []byte {
	b = e.Int(b, len(v))
	b = append(b, v...)

	return b
}

func (e *LowEncoder) BasicType(b []byte, tp byte) []byte {
	return append(b, tp)
}

func (e *LowEncoder) ResultType(b []byte, tp ...byte) []byte {
	b = e.Int(b, len(tp))
	return append(b, tp...)
}

func (e *LowEncoder) FuncType(b []byte, params, result []byte) []byte {
	b = append(b, FuncTypeHeader)
	b = e.ResultType(b, params...)
	b = e.ResultType(b, result...)

	return b
}

func (e *LowEncoder) Limits(b []byte, lo, hi int) []byte {
	if hi < 0 {
		b = append(b, LimitLo)
		return e.Int(b, lo)
	}

	b = append(b, LimitLoHi)
	b = e.Int(b, lo)
	b = e.Int(b, hi)

	return b
}

func (e *LowEncoder) TableType(b []byte, tp byte, lo, hi int) []byte {
	b = append(b, tp)
	b = e.Limits(b, lo, hi)
	return b
}

func (e *LowEncoder) GlobalType(b []byte, tp, mut byte) []byte {
	return append(b, tp, mut)
}

func (e *LowEncoder) Section(b []byte, id byte, data []byte) []byte {
	b = append(b, id)
	b = e.Int(b, len(data))
	b = append(b, data...)

	return b
}

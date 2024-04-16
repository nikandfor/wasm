package wasm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLowEncoderDecoder(tb *testing.T) {
	var (
		b []byte
		e LowEncoder
		d LowDecoder

		tp ResultType
		fn FuncType
	)

	tb.Run("Reference", func(tb *testing.T) {
		b = e.Uint64(b[:0], 624485)
		assert.Equal(tb, []byte{0xe5, 0x8e, 0x26}, b)

		b = e.Int64(b[:0], -123456)
		assert.Equal(tb, []byte{0xc0, 0xbb, 0x78}, b)
	})

	tb.Run("Unsigned", func(tb *testing.T) {
		for _, x := range []uint64{0, 1, 5, 100, 127, 128, 512, 624485, 123_456_789} {
			b = e.Uint64(b[:0], x)

			y, i, err := d.Uint64(b, 0)
			assert.NoError(tb, err)
			assert.Equal(tb, len(b), i)
			assert.Equal(tb, x, y)

			if tb.Failed() {
				tb.Logf("x: %v\nb: %x\ny: %v", x, b, y)
				break
			}
		}
	})

	tb.Run("Signed_pos", func(tb *testing.T) {
		for _, x := range []int64{0, 1, 5, 100, 127, 128, 512, 123456, 123_456_789} {
			b = e.Int64(b[:0], x)

			y, i, err := d.Int64(b, 0)
			assert.NoError(tb, err)
			assert.Equal(tb, len(b), i)
			assert.Equal(tb, x, y)

			if tb.Failed() {
				tb.Logf("x: %v\nb: %x\ny: %v", x, b, y)
				break
			}
		}
	})

	tb.Run("Signed_neg", func(tb *testing.T) {
		for _, x := range []int64{-1, -5, -100, -127, -128, -512, -123456, -123_456_789} {
			b = e.Int64(b[:0], x)

			y, i, err := d.Int64(b, 0)
			assert.NoError(tb, err)
			assert.Equal(tb, len(b), i)
			assert.Equal(tb, x, y)

			if tb.Failed() {
				tb.Logf("x: %v\nb: %x\ny: %v", x, b, y)
				break
			}
		}
	})

	tb.Run("Float", func(tb *testing.T) {
		for _, x := range []float64{0, 1, -1, 100.123456, -100.123456} {
			b = e.Float64(b[:0], x)

			y, i, err := d.Float64(b, 0)
			assert.NoError(tb, err)
			assert.Equal(tb, len(b), i)
			assert.Equal(tb, x, y)

			if tb.Failed() {
				tb.Logf("x: %v\nb: %x\ny: %v", x, b, y)
				break
			}
		}
	})

	tb.Run("Name", func(tb *testing.T) {
		for _, x := range []string{"", "1", "a", "1qaz", "Hello, 世界"} {
			b = e.Name(b[:0], x)

			y, i, err := d.Name(b, 0)
			assert.NoError(tb, err)
			assert.Equal(tb, len(b), i)
			assert.Equal(tb, x, y)

			if tb.Failed() {
				tb.Logf("x: %v\nb: %x\ny: %v", x, b, y)
				break
			}
		}
	})

	tb.Run("BasicType", func(tb *testing.T) {
		for _, x := range []byte{I32, I64, F32, F64, V128} {
			b = e.BasicType(b[:0], x)

			y, i, err := d.BasicType(b, 0)
			assert.NoError(tb, err)
			assert.Equal(tb, len(b), i)
			assert.Equal(tb, x, y)

			if tb.Failed() {
				tb.Logf("x: %v\nb: %x\ny: %v", x, b, y)
				break
			}
		}
	})

	tb.Run("ResultType", func(tb *testing.T) {
		for _, x := range []ResultType{{I32, I64}, {F32}} {
			b = e.ResultType(b[:0], x...)

			y, i, err := d.ResultType(b, 0, tp[:0])
			assert.NoError(tb, err)
			assert.Equal(tb, len(b), i)
			assert.Equal(tb, x, y)

			if tb.Failed() {
				tb.Logf("x: %v\nb: %x\ny: %v", x, b, y)
				break
			}
		}
	})

	tb.Run("FuncType", func(tb *testing.T) {
		for _, x := range []FuncType{
			{Params: ResultType{I32, I64}, Result: ResultType{F32}},
		} {
			b = e.FuncType(b[:0], x.Params, x.Result)

			y, i, err := d.FuncType(b, 0, fn)
			assert.NoError(tb, err)
			assert.Equal(tb, len(b), i)
			assert.Equal(tb, x, y)

			if tb.Failed() {
				tb.Logf("x: %v\nb: %x\ny: %v", x, b, y)
				break
			}
		}
	})

	tb.Run("Limit", func(tb *testing.T) {
		for _, x := range [][2]int{
			[2]int{0, -1},
			[2]int{1, -1},
			[2]int{0, 0},
			[2]int{0, 4},
			[2]int{1, 4},
		} {
			b = e.Limit(b[:0], x[0], x[1])

			lo, hi, i, err := d.Limit(b, 0)
			assert.NoError(tb, err)
			assert.Equal(tb, len(b), i)
			assert.Equal(tb, x, [2]int{lo, hi})

			if tb.Failed() {
				tb.Logf("x: %v\nb: %x\ny: %v", x, b, [2]int{lo, hi})
				break
			}
		}
	})

	tb.Run("TableType", func(tb *testing.T) {
		type TableType struct {
			tp     byte
			lo, hi int
		}

		for _, x := range []TableType{
			{tp: I64, lo: 0, hi: 5},
			{tp: F32, lo: 4, hi: -1},
		} {
			b = e.TableType(b[:0], x.tp, x.lo, x.hi)

			tp, lo, hi, i, err := d.TableType(b, 0)
			assert.NoError(tb, err)
			assert.Equal(tb, len(b), i)
			assert.Equal(tb, x, TableType{tp: tp, lo: lo, hi: hi})

			if tb.Failed() {
				tb.Logf("x: %v\nb: %x\ny: %v", x, b, TableType{tp: tp, lo: lo, hi: hi})
				break
			}
		}
	})

	tb.Run("GlobalType", func(tb *testing.T) {
		type GlobalType struct {
			tp, mut byte
		}

		for _, x := range []GlobalType{
			{tp: I64, mut: 1},
			{tp: F32, mut: 0},
		} {
			b = e.GlobalType(b[:0], x.tp, x.mut)

			tp, mut, i, err := d.GlobalType(b, 0)
			assert.NoError(tb, err)
			assert.Equal(tb, len(b), i)
			assert.Equal(tb, x, GlobalType{tp: tp, mut: mut})

			if tb.Failed() {
				tb.Logf("x: %v\nb: %x\ny: %v", x, b, GlobalType{tp: tp, mut: mut})
				break
			}
		}
	})
}

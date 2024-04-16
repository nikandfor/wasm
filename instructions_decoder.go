package wasm

import (
	"fmt"

	"tlog.app/go/errors"
	"tlog.app/go/tlog"
)

type (
	InstructionsDecoder struct {
		LowDecoder
	}

	UnsupportedOpcodeError struct {
		Opcode Opcode
		Args   []byte
	}

	Opcode byte
)

// Opcodes
const (
	Unreachable = 0x00
	Nop         = 0x01

	Block   = 0x02
	Loop    = 0x03
	If      = 0x04
	Else    = 0x05
	End     = 0x0b
	Br      = 0x0c
	BrIf    = 0x0d
	BrTable = 0x0e
	Ret     = 0x0f

	Call      = 0x10
	CallIndir = 0x11

	Drop   = 0x1a
	Select = 0x1b

	LocalGet  = 0x20
	LocalSet  = 0x21
	LocalTee  = 0x22
	GlobalGet = 0x23
	GlobalSet = 0x24

	I32Load    = 0x28
	I64Load    = 0x29
	F32Load    = 0x2a
	F64Load    = 0x2b
	I32Load8S  = 0x2c
	I32Load8U  = 0x2d
	I32Load16S = 0x2e
	I32Load16U = 0x2f
	I64Load8S  = 0x30
	I64Load8U  = 0x31
	I64Load16S = 0x32
	I64Load16U = 0x33
	I64Load32S = 0x34
	I64Load32U = 0x35
	I32Store   = 0x36
	I64Store   = 0x37
	F32Store   = 0x38
	F64Store   = 0x39
	I32Store8  = 0x3a
	I32Store16 = 0x3b
	I64Store8  = 0x3c
	I64Store16 = 0x3d
	I64Store32 = 0x3e

	MemorySize = 0x3f
	MemoryGrow = 0x40

	I32Const = 0x41
	I64Const = 0x42
	F32Const = 0x43
	F64Const = 0x44

	I32EqZ = 0x45
	I32Eq  = 0x46
	I32Ne  = 0x47
	I32LtS = 0x48
	I32LtU = 0x49
	I32GtS = 0x4a
	I32GtU = 0x4b
	I32LeS = 0x4c
	I32LeU = 0x4d
	I32GeS = 0x4e
	I32GeU = 0x4f

	I64EqZ = 0x50
	I64Eq  = 0x51
	I64Ne  = 0x52
	I64LtS = 0x53
	I64LtU = 0x54
	I64GtS = 0x55
	I64GtU = 0x56
	I64LeS = 0x57
	I64LeU = 0x58
	I64GeS = 0x59
	I64GeU = 0x5a

	F32Eq = 0x5b
	F32Ne = 0x5c
	F32Lt = 0x5d
	F32Gt = 0x5e
	F32Le = 0x5f
	F32Ge = 0x60

	F64Eq = 0x61
	F64Ne = 0x62
	F64Lt = 0x63
	F64Gt = 0x64
	F64Le = 0x65
	F64Ge = 0x66

	I32Clz    = 0x67
	I32Ctz    = 0x68
	I32Popcnt = 0x69
	I32Add    = 0x6a
	I32Sub    = 0x6b
	I32Mul    = 0x6c
	I32DivS   = 0x6d
	I32DivU   = 0x6e
	I32RemS   = 0x6f
	I32RemU   = 0x70
	I32And    = 0x71
	I32Or     = 0x72
	I32Xor    = 0x73
	I32Shl    = 0x74
	I32ShrS   = 0x75
	I32ShrU   = 0x76
	I32RotL   = 0x77
	I32RotR   = 0x78

	I64Clz    = 0x79
	I64Ctz    = 0x7a
	I64Popcnt = 0x7b
	I64Add    = 0x7c
	I64Sub    = 0x7d
	I64Mul    = 0x7e
	I64DivS   = 0x7f
	I64DivU   = 0x80
	I64RemS   = 0x81
	I64RemU   = 0x82
	I64And    = 0x83
	I64Or     = 0x84
	I64Xor    = 0x85
	I64Shl    = 0x86
	I64ShrS   = 0x87
	I64ShrU   = 0x88
	I64RotL   = 0x89
	I64RotR   = 0x8a

	F32Abs      = 0x8b
	F32Neg      = 0x8c
	F32Ceil     = 0x8d
	F32Floor    = 0x8e
	F32Trunc    = 0x8f
	F32Near     = 0x90
	F32Sqrt     = 0x91
	F32Add      = 0x92
	F32Sub      = 0x93
	F32Mul      = 0x94
	F32Div      = 0x95
	F32Min      = 0x96
	F32Max      = 0x97
	F32CopySign = 0x98

	F64Abs      = 0x99
	F64Neg      = 0x9a
	F64Ceil     = 0x9b
	F64Floor    = 0x9c
	F64Trunc    = 0x9d
	F64Near     = 0x9e
	F64Sqrt     = 0x9f
	F64Add      = 0xa0
	F64Sub      = 0xa1
	F64Mul      = 0xa2
	F64Div      = 0xa3
	F64Min      = 0xa4
	F64Max      = 0xa5
	F64CopySign = 0xa6

	FCExt = 0xfc
)

// FC ext opcodes
const (
	FCMemoryCopy = 0x0a
	FCMemoryFill = 0x0b
)

func (d *InstructionsDecoder) Expr(b []byte, st int) (code []byte, i int, err error) {
	i = st
	depth := 0

	for i < len(b) {
		opst := i
		op := Opcode(b[i])
		i++

		switch {
		case op <= Nop || op == Else || op == Ret:
		case op == Block || op == Loop || op == If:
			depth++
		case op == End:
			depth--
		case op == Br || op == BrIf:
			_, i, err = d.Int64(b, i)
		case op == BrTable:
			var l int
			l, i, err = d.Int(b, i)
			if err != nil {
				return nil, opst, err
			}

			for j := 0; j < l+1; j++ {
				_, i, err = d.Int(b, i)
				if err != nil {
					return nil, opst, err
				}
			}
		case op == Call:
			_, i, err = d.Int64(b, i)
		case op == CallIndir:
			_, i, err = d.Int64(b, i)
			i++
		case op == Drop || op == Select:
		case op >= LocalGet && op <= GlobalSet:
			_, i, err = d.Int64(b, i)
		case op >= I32Load && op <= I64Store32:
			_, i, err = d.Int64(b, i)
			if err != nil {
				return nil, opst, err
			}

			_, i, err = d.Int64(b, i)
			if err != nil {
				return nil, opst, err
			}
		case op == MemorySize || op == MemoryGrow:
		case op >= I32Const && op <= F64Const:
			_, i, err = d.Int64(b, i)
		case op >= I32EqZ && op <= F64CopySign:
		case op == FCExt:
			i, err = d.fcExt(b, opst)
		default:
			err = UnsupportedOpcodeError{Opcode: op}
			err = errors.Wrap(err, "at pos 0x%x", opst)
		}

		tlog.V("opcode").Printw("opcode", "i", tlog.NextAsHex, opst, "op", op, "code", tlog.NextAsHex, b[opst:i])

		if err != nil {
			return nil, opst, err
		}

		if depth < 0 {
			return b[st:i], i, nil
		}
	}

	return nil, st, ErrUnexpectedEOF
}

func (d *InstructionsDecoder) Func(b []byte, buf FuncCode) (f FuncCode, err error) {
	f = buf

	l, i, err := d.Int(b, 0)
	if err != nil {
		return f, err
	}

	f.Locals = f.Locals[:0]

	var cnt int
	var tp byte

	for n := 0; n < l; n++ {
		cnt, i, err = d.Int(b, i)
		if err != nil {
			return f, err
		}

		tp, i, err = d.Byte(b, i)
		if err != nil {
			return f, err
		}

		for j := 0; j < cnt; j++ {
			f.Locals = append(f.Locals, Type(tp))
		}
	}

	f.Expr, i, err = d.Expr(b, i)
	if err != nil {
		return f, errors.Wrap(err, "expr")
	}

	if i != len(b) {
		return f, ErrSizeMismatch
	}

	return f, nil
}

func (d *InstructionsDecoder) fcExt(b []byte, st int) (i int, err error) {
	op, i, err := d.Byte(b, st)
	if err != nil {
		return
	}
	if op != FCExt {
		return st, errors.New("fc ext expected")
	}

	op, i, err = d.Byte(b, i)
	if err != nil {
		return
	}

	switch op {
	case FCMemoryCopy, FCMemoryFill:
	default:
		return st, UnsupportedOpcodeError{Opcode: FCExt, Args: b[i-1 : i]}
	}

	return i, nil
}

func (e UnsupportedOpcodeError) Error() string {
	return fmt.Sprintf("unsupported opcode: %v [% 02x]", e.Opcode, e.Args)
}

func (op Opcode) String() string {
	if n := opNames[op]; n != "" {
		return n
	}

	return fmt.Sprintf("%02x", int(op))
}

var opNames = [...]string{
	Unreachable: "Unreachable",
	Nop:         "Nop",

	Block:   "Block",
	Loop:    "Loop",
	If:      "If",
	Else:    "Else",
	End:     "End",
	Br:      "Br",
	BrIf:    "BrIf",
	BrTable: "BrTable",
	Ret:     "Ret",

	LocalGet:  "LocalGet",
	LocalSet:  "LocalSet",
	LocalTee:  "LocalTee",
	GlobalGet: "GlobalGet",
	GlobalSet: "GlobalSet",

	I32Load:    "I32Load",
	I64Load:    "I64Load",
	F32Load:    "F32Load",
	F64Load:    "F64Load",
	I32Load8S:  "I32Load8S",
	I32Load8U:  "I32Load8U",
	I32Load16S: "I32Load16S",
	I32Load16U: "I32Load16U",
	I64Load8S:  "I64Load8S",
	I64Load8U:  "I64Load8U",
	I64Load16S: "I64Load16S",
	I64Load16U: "I64Load16U",
	I64Load32S: "I64Load32S",
	I64Load32U: "I64Load32U",
	I32Store:   "I32Store",
	I64Store:   "I64Store",
	F32Store:   "F32Store",
	F64Store:   "F64Store",
	I32Store8:  "I32Store8",
	I32Store16: "I32Store16",
	I64Store8:  "I64Store8",
	I64Store16: "I64Store16",
	I64Store32: "I64Store32",

	MemorySize: "MemorySize",
	MemoryGrow: "MemoryGrow",

	I32Const: "I32Const",
	I64Const: "I64Const",
	F32Const: "F32Const",
	F64Const: "F64Const",

	I32EqZ: "I32EqZ",
	I32Eq:  "I32Eq",
	I32Ne:  "I32Ne",
	I32LtS: "I32LtS",
	I32LtU: "I32LtU",
	I32GtS: "I32GtS",
	I32GtU: "I32GtU",
	I32LeS: "I32LeS",
	I32LeU: "I32LeU",
	I32GeS: "I32GeS",
	I32GeU: "I32GeU",

	I64EqZ: "I64EqZ",
	I64Eq:  "I64Eq",
	I64Ne:  "I64Ne",
	I64LtS: "I64LtS",
	I64LtU: "I64LtU",
	I64GtS: "I64GtS",
	I64GtU: "I64GtU",
	I64LeS: "I64LeS",
	I64LeU: "I64LeU",
	I64GeS: "I64GeS",
	I64GeU: "I64GeU",

	F32Eq: "F32Eq",
	F32Ne: "F32Ne",
	F32Lt: "F32Lt",
	F32Gt: "F32Gt",
	F32Le: "F32Le",
	F32Ge: "F32Ge",

	F64Eq: "F64Eq",
	F64Ne: "F64Ne",
	F64Lt: "F64Lt",
	F64Gt: "F64Gt",
	F64Le: "F64Le",
	F64Ge: "F64Ge",

	I32Clz:    "I32Clz",
	I32Ctz:    "I32Ctz",
	I32Popcnt: "I32Popcnt",
	I32Add:    "I32Add",
	I32Sub:    "I32Sub",
	I32Mul:    "I32Mul",
	I32DivS:   "I32DivS",
	I32DivU:   "I32DivU",
	I32RemS:   "I32RemS",
	I32RemU:   "I32RemU",
	I32And:    "I32And",
	I32Or:     "I32Or",
	I32Xor:    "I32Xor",
	I32Shl:    "I32Shl",
	I32ShrS:   "I32ShrS",
	I32ShrU:   "I32ShrU",
	I32RotL:   "I32RotL",
	I32RotR:   "I32RotR",

	I64Clz:    "I64Clz",
	I64Ctz:    "I64Ctz",
	I64Popcnt: "I64Popcnt",
	I64Add:    "I64Add",
	I64Sub:    "I64Sub",
	I64Mul:    "I64Mul",
	I64DivS:   "I64DivS",
	I64DivU:   "I64DivU",
	I64RemS:   "I64RemS",
	I64RemU:   "I64RemU",
	I64And:    "I64And",
	I64Or:     "I64Or",
	I64Xor:    "I64Xor",
	I64Shl:    "I64Shl",
	I64ShrS:   "I64ShrS",
	I64ShrU:   "I64ShrU",
	I64RotL:   "I64RotL",
	I64RotR:   "I64RotR",

	F32Abs:      "F32Abs",
	F32Neg:      "F32Neg",
	F32Ceil:     "F32Ceil",
	F32Floor:    "F32Floor",
	F32Trunc:    "F32Trunc",
	F32Near:     "F32Near",
	F32Sqrt:     "F32Sqrt",
	F32Add:      "F32Add",
	F32Sub:      "F32Sub",
	F32Mul:      "F32Mul",
	F32Div:      "F32Div",
	F32Min:      "F32Min",
	F32Max:      "F32Max",
	F32CopySign: "F32CopySign",

	F64Abs:      "F64Abs",
	F64Neg:      "F64Neg",
	F64Ceil:     "F64Ceil",
	F64Floor:    "F64Floor",
	F64Trunc:    "F64Trunc",
	F64Near:     "F64Near",
	F64Sqrt:     "F64Sqrt",
	F64Add:      "F64Add",
	F64Sub:      "F64Sub",
	F64Mul:      "F64Mul",
	F64Div:      "F64Div",
	F64Min:      "F64Min",
	F64Max:      "F64Max",
	F64CopySign: "F64CopySign",

	255: "",
}

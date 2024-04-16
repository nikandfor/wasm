package wasm

import "tlog.app/go/tlog/tlwire"

type (
	Module struct {
		Version int

		DataCount int
		Start     Index

		Type     []FuncType
		Import   []Import
		Function []Index
		Table    []Table
		Memory   []Limits
		Global   []Global
		Export   []Export
		Element  []Element
		Code     []Code
		Data     []Data

		Custom []Custom

		Sections []byte
	}

	Index int
	Type  byte
	Code  []byte

	ResultType []Type

	FuncType struct {
		Params ResultType
		Result ResultType
	}

	Import struct {
		Module, Name []byte

		// raw storage for import description
		// tp 0 => typeidx at rawi[0]
		// tp 1 => refype  at rawb[0], lo, hi limits at rawi
		// tp 2 => memtype at rawi
		// tp 3 => valtype at rawb[0], mut at rawb[1]
		tp   byte
		rawb [2]byte
		rawi [2]int
	}

	Export struct {
		Name []byte

		ExportType byte

		Index Index
	}

	Table struct {
		Type   Type
		Limits Limits
	}

	Limits struct {
		Lo, Hi int
	}

	Global struct {
		Type Type
		Mut  byte
		Expr Code
	}

	Element struct {
		Type Type
		Expr Code

		Funcs []Index
	}

	Data struct {
		Expr Code
		Init []byte
	}

	Custom struct {
		Name []byte
		Data []byte
	}

	FuncCode struct {
		Locals ResultType
		Expr   Code
	}
)

// Basic types.
const (
	I32 = 0x7f
	I64 = 0x7e
	F32 = 0x7d
	F64 = 0x7c

	V128 = 0x7b

	FuncRef   = 0x70
	ExternRef = 0x6f

	FuncTypeHeader = 0x60

	LimitLo   = 0x00
	LimitLoHi = 0x01
)

// Section ids.
const (
	CustomSection = iota
	TypeSection
	ImportSection
	FunctionSection
	TableSection
	MemorySection
	GlobalSection
	ExportSection
	StartSection
	ElementSection
	CodeSection
	DataSection
	DataCountSection

	sectionNext
)

func init() {
	if sectionNext != 13 {
		panic(sectionNext)
	}
}

func (c Code) TlogAppend(b []byte) []byte {
	var e tlwire.Encoder

	b = e.AppendSemantic(b, tlwire.Hex)

	return e.AppendBytes(b, c)

	b = e.AppendArray(b, len(c))

	for _, v := range c {
		b = e.AppendInt(b, int(v))
	}

	return b
}

func (tp ResultType) TlogAppend(b []byte) []byte {
	var e tlwire.Encoder

	b = e.AppendSemantic(b, tlwire.Hex)
	b = e.AppendArray(b, len(tp))

	for _, t := range tp {
		b = e.AppendInt(b, int(t))
	}

	return b
}

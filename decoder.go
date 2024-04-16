package wasm

import (
	"encoding/binary"
	stderrors "errors"
	"io"
	"math"
	"unsafe"

	"tlog.app/go/errors"
)

type (
	Decoder struct {
		InstructionsDecoder
	}

	LowDecoder struct {
		//	Copy bool
	}
)

var (
	Magic               = []byte("\000asm")
	MaxSupportedVersion = 1
)

var (
	ErrMagic              = stderrors.New("magic mismatch")
	ErrOverflow           = stderrors.New("integer overflow")
	ErrSizeMismatch       = stderrors.New("size mismatch")
	ErrUnexpectedEOF      = io.ErrUnexpectedEOF
	ErrUnsupportedVersion = stderrors.New("unsupported binary format version")
)

func (d *Decoder) Module(b []byte, m *Module) (err error) {
	i := 0

	defer func() {
		if err == nil {
			return
		}

		err = errors.Wrap(err, "at pos 0x%x", i)
	}()

	if common(b[i:], Magic) != len(Magic) {
		return ErrMagic
	}

	i += len(Magic)

	if i+4 > len(b) {
		return ErrUnexpectedEOF
	}

	m.Version = int(binary.LittleEndian.Uint32(b[i:]))
	i += 4

	if m.Version > MaxSupportedVersion {
		return ErrUnsupportedVersion
	}

	m.Start = -1
	m.DataCount = 0

	for i < len(b) {
		id := b[i]
		m.Sections = append(m.Sections, id)

		size, end, err := d.Int(b, i+1)
		if err != nil {
			return errors.Wrap(err, "section size")
		}

		end += size
		if end > len(b) {
			return ErrUnexpectedEOF
		}

		switch id {
		case CustomSection:
			i, err = d.CustomSection(b, i, m)
		case TypeSection:
			i, err = d.TypeSection(b, i, m)
		case ImportSection:
			i, err = d.ImportSection(b, i, m)
		case FunctionSection:
			i, err = d.FunctionSection(b, i, m)
		case TableSection:
			i, err = d.TableSection(b, i, m)
		case MemorySection:
			i, err = d.MemorySection(b, i, m)
		case GlobalSection:
			i, err = d.GlobalSection(b, i, m)
		case ExportSection:
			i, err = d.ExportSection(b, i, m)
		case ElementSection:
			i, err = d.ElementSection(b, i, m)
		case CodeSection:
			i, err = d.CodeSection(b, i, m)
		case DataSection:
			i, err = d.DataSection(b, i, m)
		case DataCountSection:
			i, err = d.DataCountSection(b, i, m)
		case StartSection:
			i, err = d.StartSection(b, i, m)
		default:
			return errors.New("unsupported section id: 0x%02x", id)
		}

		if err != nil {
			return errors.Wrap(err, "section id %x", id)
		}

		i = end
	}

	return nil
}

func (d *Decoder) CustomSection(b []byte, st int, m *Module) (i int, err error) {
	end, i, err := d.sectionHeader(b, st, CustomSection)
	if err != nil {
		return i, err
	}

	name, i, err := d.Name(b, i)
	if err != nil {
		return st, errors.Wrap(err, "name")
	}

	data := b[i:end]

	n := len(m.Custom)

	if len(m.Custom) < cap(m.Custom) {
		m.Custom = m.Custom[:n+1]
	} else {
		m.Custom = append(m.Custom, Custom{})
	}

	m.Custom[n].Name = appendOrSet(false, m.Custom[n].Name[:0], name...)
	m.Custom[n].Data = appendOrSet(false, m.Custom[n].Data[:0], data...)

	return end, nil
}

func (d *Decoder) TypeSection(b []byte, st int, m *Module) (i int, err error) {
	l, end, i, err := d.sectionHeaderLen(b, st, TypeSection)
	if err != nil {
		return i, err
	}

	m.Type = m.Type[:cap(m.Type)]

	for n := 0; n < l; n++ {
		for n >= len(m.Type) {
			m.Type = append(m.Type, FuncType{})
		}

		m.Type[n], i, err = d.FuncType(b, i, m.Type[n])
		if err != nil {
			return i, errors.Wrap(err, "func %d", n)
		}
	}

	m.Type = m.Type[:l]

	if i != end {
		return st, ErrSizeMismatch
	}

	return i, nil
}

func (d *Decoder) ImportSection(b []byte, st int, m *Module) (i int, err error) {
	l, end, i, err := d.sectionHeaderLen(b, st, ImportSection)
	if err != nil {
		return i, err
	}

	m.Import = m.Import[:cap(m.Import)]

	for n := 0; n < l; n++ {
		for n >= len(m.Import) {
			m.Import = append(m.Import, Import{})
		}

		m.Import[n], i, err = d.Import(b, i, m.Import[n])
		if err != nil {
			return i, errors.Wrap(err, "import %d", n)
		}
	}

	m.Import = m.Import[:l]

	if i != end {
		return st, ErrSizeMismatch
	}

	return i, nil
}

func (d *Decoder) Import(b []byte, st int, buf Import) (im Import, i int, err error) {
	im = buf
	i = st

	x, i, err := d.Name(b, i)
	if err != nil {
		return im, i, errors.Wrap(err, "module")
	}

	im.Module = appendOrSet(false, im.Module[:0], x...)

	x, i, err = d.Name(b, i)
	if err != nil {
		return im, i, errors.Wrap(err, "name")
	}

	im.Name = appendOrSet(false, im.Name[:0], x...)

	if i+2 > len(b) {
		return im, i, ErrUnexpectedEOF
	}

	im.tp = b[i]
	i++

	switch im.tp {
	case 0:
		im.rawi[0], i, err = d.Int(b, i)
		if err != nil {
			return im, i, errors.Wrap(err, "type index")
		}
	case 1:
		im.rawb[0], im.rawi[0], im.rawi[1], i, err = d.TableType(b, i)
		if err != nil {
			return im, i, errors.Wrap(err, "table type")
		}
	case 2:
		im.rawi[0], im.rawi[1], i, err = d.Limits(b, i)
		if err != nil {
			return im, i, errors.Wrap(err, "memory limits")
		}
	case 3:
		im.rawb[0], im.rawb[1], i, err = d.GlobalType(b, i)
		if err != nil {
			return im, i, errors.Wrap(err, "global type")
		}
	default:
		return im, i - 1, errors.New("unsupported import description type: 0x%02x", im.tp)
	}

	return im, i, nil
}

func (d *Decoder) ExportSection(b []byte, st int, m *Module) (i int, err error) {
	l, end, i, err := d.sectionHeaderLen(b, st, ExportSection)
	if err != nil {
		return i, err
	}

	m.Export = m.Export[:cap(m.Export)]

	for n := 0; n < l; n++ {
		for n >= len(m.Export) {
			m.Export = append(m.Export, Export{})
		}

		m.Export[n], i, err = d.Export(b, i, m.Export[n])
		if err != nil {
			return i, errors.Wrap(err, "export %d", n)
		}
	}

	m.Export = m.Export[:l]

	if i != end {
		return st, ErrSizeMismatch
	}

	return i, nil
}

func (d *Decoder) Export(b []byte, st int, buf Export) (ex Export, i int, err error) {
	ex = buf
	i = st

	x, i, err := d.Name(b, i)
	if err != nil {
		return ex, i, errors.Wrap(err, "name")
	}

	ex.Name = appendOrSet(false, ex.Name[:0], x...)

	if i+2 > len(b) {
		return ex, i, ErrUnexpectedEOF
	}

	ex.ExportType = b[i]
	i++

	idx, i, err := d.Int(b, i)
	if err != nil {
		return ex, st, errors.Wrap(err, "index")
	}

	ex.Index = Index(idx)

	return ex, i, nil
}

func (d *Decoder) FunctionSection(b []byte, st int, m *Module) (i int, err error) {
	l, end, i, err := d.sectionHeaderLen(b, st, FunctionSection)
	if err != nil {
		return i, err
	}

	var f int
	m.Function = m.Function[:0]

	for n := 0; n < l; n++ {
		f, i, err = d.Int(b, i)
		if err != nil {
			return i, errors.Wrap(err, "func %d", n)
		}

		m.Function = append(m.Function, Index(f))
	}

	if i != end {
		return st, ErrSizeMismatch
	}

	return i, nil
}

func (d *Decoder) TableSection(b []byte, st int, m *Module) (i int, err error) {
	l, end, i, err := d.sectionHeaderLen(b, st, TableSection)
	if err != nil {
		return i, err
	}

	var x Table
	var tp byte
	m.Table = m.Table[:0]

	for n := 0; n < l; n++ {
		tp, x.Limits.Lo, x.Limits.Hi, i, err = d.TableType(b, i)
		if err != nil {
			return i, errors.Wrap(err, "table %d", n)
		}

		x.Type = Type(tp)

		m.Table = append(m.Table, x)
	}

	if i != end {
		return st, ErrSizeMismatch
	}

	return i, nil
}

func (d *Decoder) MemorySection(b []byte, st int, m *Module) (i int, err error) {
	l, end, i, err := d.sectionHeaderLen(b, st, MemorySection)
	if err != nil {
		return i, err
	}

	var x Limits
	m.Memory = m.Memory[:0]

	for n := 0; n < l; n++ {
		x.Lo, x.Hi, i, err = d.Limits(b, i)
		if err != nil {
			return i, errors.Wrap(err, "memory %d", n)
		}

		m.Memory = append(m.Memory, x)
	}

	if i != end {
		return st, ErrSizeMismatch
	}

	return i, nil
}

func (d *Decoder) GlobalSection(b []byte, st int, m *Module) (i int, err error) {
	l, end, i, err := d.sectionHeaderLen(b, st, GlobalSection)
	if err != nil {
		return i, err
	}

	var tp, mut byte
	var code Code
	m.Global = m.Global[:0]

	for n := 0; n < l; n++ {
		for n >= len(m.Global) {
			m.Global = append(m.Global, Global{})
		}

		tp, mut, i, err = d.GlobalType(b, i)
		if err != nil {
			return i, errors.Wrap(err, "global type %d", n)
		}

		m.Global[n].Type = Type(tp)
		m.Global[n].Mut = mut

		code, i, err = d.Expr(b, i)
		if err != nil {
			return i, errors.Wrap(err, "global code %d", n)
		}

		m.Global[n].Expr = appendOrSet(false, m.Global[n].Expr[:0], code...)
	}

	m.Global = m.Global[:l]

	if i != end {
		return st, ErrSizeMismatch
	}

	return i, nil
}

func (d *Decoder) ElementSection(b []byte, st int, m *Module) (i int, err error) {
	l, end, i, err := d.sectionHeaderLen(b, st, ElementSection)
	if err != nil {
		return i, err
	}

	m.Element = m.Element[:cap(m.Element)]

	for n := 0; n < l; n++ {
		for n >= len(m.Element) {
			m.Element = append(m.Element, Element{})
		}

		m.Element[n], i, err = d.Element(b, i, m.Element[n])
		if err != nil {
			return i, errors.Wrap(err, "element %d", n)
		}
	}

	m.Element = m.Element[:l]

	if i != end {
		return st, ErrSizeMismatch
	}

	return i, nil
}

func (d *Decoder) Element(b []byte, st int, buf Element) (el Element, i int, err error) {
	el = buf

	tp, i, err := d.Byte(b, st)
	if err != nil {
		return el, i, err
	}

	switch tp {
	case 0:
		var code []byte

		code, i, err = d.Expr(b, i)
		if err != nil {
			return el, i, errors.Wrap(err, "expr")
		}

		el.Expr = appendOrSet(false, el.Expr[:0], code...)

		var l int

		l, i, err = d.Int(b, i)
		if err != nil {
			return el, i, errors.Wrap(err, "funcs")
		}

		for j := 0; j < l; j++ {
			var id int
			id, i, err = d.Int(b, i)
			if err != nil {
				return el, i, errors.Wrap(err, "func id")
			}

			el.Funcs = append(el.Funcs, Index(id))
		}
	default:
		return el, st, errors.New("unsupported elem: 0x%02x", tp)
	}

	return el, i, nil
}

func (d *Decoder) DataCountSection(b []byte, st int, m *Module) (i int, err error) {
	l, end, i, err := d.sectionHeaderLen(b, st, DataCountSection)
	if err != nil {
		return
	}

	m.DataCount = l

	if i != end {
		return st, ErrSizeMismatch
	}

	return
}

func (d *Decoder) StartSection(b []byte, st int, m *Module) (i int, err error) {
	l, end, i, err := d.sectionHeaderLen(b, st, StartSection)
	if err != nil {
		return
	}

	m.Start = Index(l)

	if i != end {
		return st, ErrSizeMismatch
	}

	return
}

func (d *Decoder) CodeSection(b []byte, st int, m *Module) (i int, err error) {
	l, end, i, err := d.sectionHeaderLen(b, st, CodeSection)
	if err != nil {
		return
	}

	var size int
	m.Code = m.Code[:cap(m.Code)]

	for n := 0; n < l; n++ {
		for n >= len(m.Code) {
			m.Code = append(m.Code, nil)
		}

		size, i, err = d.Int(b, i)
		if err != nil {
			return i, errors.Wrap(err, "code %d", n)
		}

		if i+size > len(b) {
			return i, errors.Wrap(ErrUnexpectedEOF, "code %d", n)
		}

		m.Code[n] = appendOrSet(false, m.Code[n][:0], b[i:i+size]...)
		i += size
	}

	m.Code = m.Code[:l]

	if i != end {
		return st, ErrSizeMismatch
	}

	return
}

func (d *Decoder) DataSection(b []byte, st int, m *Module) (i int, err error) {
	l, end, i, err := d.sectionHeaderLen(b, st, DataSection)
	if err != nil {
		return
	}

	var size int
	var code []byte
	var tp byte
	m.Data = m.Data[:cap(m.Data)]

	for n := 0; n < l; n++ {
		for n >= len(m.Data) {
			m.Data = append(m.Data, Data{})
		}

		tp, i, err = d.Byte(b, i)
		if err != nil {
			return
		}

		switch tp {
		case 0:
			code, i, err = d.Expr(b, i)
			if err != nil {
				return i, errors.Wrap(err, "data %d: expr", n)
			}

			m.Data[n].Expr = appendOrSet(false, m.Data[n].Expr[:0], code...)
		default:
			return st, errors.New("data %d: unsupported data type: 0x%02x", n, tp)
		}

		size, i, err = d.Int(b, i)
		if err != nil {
			return st, errors.Wrap(err, "data %d: init size", n)
		}

		if i+size > len(b) {
			return st, errors.Wrap(ErrUnexpectedEOF, "data %d: init", n)
		}

		m.Data[n].Init = appendOrSet(false, m.Data[n].Init[:0], b[i:i+size]...)
		i += size
	}

	m.Data = m.Data[:l]

	if i != end {
		return st, ErrSizeMismatch
	}

	return
}

func (d *Decoder) sectionHeader(b []byte, st int, section byte) (end, i int, err error) {
	tp, i, err := d.Byte(b, st)
	if err != nil {
		return
	}

	if tp != section {
		return 0, st, errors.New("unexpected section id: %d, wanted %d", tp, section)
	}

	size, i, err := d.Int(b, i)
	if err != nil {
		return 0, i, errors.Wrap(err, "section size")
	}

	return i + size, i, nil
}

func (d *Decoder) sectionHeaderLen(b []byte, st int, section byte) (l, end, i int, err error) {
	end, i, err = d.sectionHeader(b, st, section)
	if err != nil {
		return 0, 0, i, err
	}

	l, i, err = d.Int(b, i)
	if err != nil {
		return 0, 0, i, errors.Wrap(err, "vector length")
	}

	return l, end, i, nil
}

func (d *LowDecoder) Byte(b []byte, st int) (r byte, i int, err error) {
	i = st

	if i >= len(b) {
		return 0, i, ErrUnexpectedEOF
	}

	return b[i], i + 1, nil
}

func (d *LowDecoder) Int(b []byte, st int) (l, i int, err error) {
	x, i, err := d.Uint64(b, st)
	return int(x), i, err
}

func (d *LowDecoder) Uint64(b []byte, st int) (v uint64, i int, err error) {
	var s uint
	i = st

	for i < len(b) {
		v |= uint64(b[i]&0x7f) << s
		i++
		s += 7

		if b[i-1]&0x80 == 0 {
			return v, i, nil
		}

		if s+7 > 64 {
			return v, st, ErrOverflow
		}
	}

	return 0, st, ErrUnexpectedEOF
}

func (d *LowDecoder) Int64(b []byte, st int) (v int64, i int, err error) {
	var s uint
	i = st

	for i < len(b) {
		v |= int64(b[i]&0x7f) << s
		i++
		s += 7

		if b[i-1]&0x80 == 0 {
			v = v << (64 - s) >> (64 - s)

			return v, i, nil
		}

		if s+7 > 64 {
			return v, st, ErrOverflow
		}
	}

	return 0, st, ErrUnexpectedEOF
}

func (d *LowDecoder) Float64(b []byte, st int) (v float64, i int, err error) {
	if st+8 > len(b) {
		return 0, st, ErrUnexpectedEOF
	}

	x := uint64(b[i]) | uint64(b[i+1])<<8 | uint64(b[i+2])<<16 | uint64(b[i+3])<<24 | uint64(b[i+4])<<32 | uint64(b[i+5])<<40 | uint64(b[i+6])<<48 | uint64(b[i+7])<<56

	return math.Float64frombits(x), st + 8, nil
}

func (d *LowDecoder) NameString(b []byte, st int) (v string, i int, err error) {
	r, i, err := d.Name(b, st)

	return string(r), i, err
}

func (d *LowDecoder) Name(b []byte, st int) (v []byte, i int, err error) {
	l, i, err := d.Int(b, st)
	if err != nil {
		return nil, st, err
	}

	if i+l > len(b) {
		return nil, st, ErrUnexpectedEOF
	}

	v = b[i : i+l]
	i += l

	return v, i, nil
}

func (d *LowDecoder) BasicType(b []byte, st int) (tp byte, i int, err error) {
	if st == len(b) {
		return 0, st, ErrUnexpectedEOF
	}

	return b[st], st + 1, nil
}

func (d *LowDecoder) ResultType(b []byte, st int, buf ResultType) (tp ResultType, i int, err error) {
	tp = buf

	l, i, err := d.Int(b, st)
	if err != nil {
		return tp, st, err
	}

	if i+l > len(b) {
		return tp, st, ErrUnexpectedEOF
	}

	//	buf = appendOrSet(false, buf, b[i:i+l]...)
	r := b[i : i+l]
	i += l

	r1 := *(*ResultType)(unsafe.Pointer(&r))
	tp = appendOrSet(false, tp, r1...)

	return tp, i, nil
}

func (d *LowDecoder) FuncType(b []byte, st int, buf FuncType) (fn FuncType, i int, err error) {
	i = st
	fn = buf

	if i+3 > len(b) {
		return buf, st, ErrUnexpectedEOF
	}

	if b[i] != FuncTypeHeader {
		return buf, st, errors.New("expected function type, got 0x%2x", b[i])
	}
	i++

	fn.Params, i, err = d.ResultType(b, i, fn.Params[:0])
	if err != nil {
		return buf, i, errors.Wrap(err, "func params")
	}

	fn.Result, i, err = d.ResultType(b, i, fn.Result[:0])
	if err != nil {
		return fn, i, errors.Wrap(err, "func result")
	}

	return fn, i, nil
}

func (d *LowDecoder) Limits(b []byte, st int) (lo, hi, i int, err error) {
	tp, i, err := d.Byte(b, st)
	if err != nil {
		return
	}

	if tp != LimitLo && tp != LimitLoHi {
		i = st
		err = errors.New("expected limit, got 0x%02x", tp)
		return
	}

	lo, i, err = d.Int(b, i)
	if err != nil {
		return
	}

	if tp == LimitLo {
		hi = -1
		return
	}

	hi, i, err = d.Int(b, i)
	if err != nil {
		return
	}

	return
}

func (d *LowDecoder) TableType(b []byte, st int) (tp byte, lo, hi, i int, err error) {
	i = st

	if i+3 > len(b) {
		err = ErrUnexpectedEOF
		return
	}

	tp, i, err = d.BasicType(b, i)
	if err != nil {
		return
	}

	lo, hi, i, err = d.Limits(b, i)
	if err != nil {
		return
	}

	return
}

func (d *LowDecoder) GlobalType(b []byte, st int) (tp, mut byte, i int, err error) {
	i = st

	if i+2 > len(b) {
		err = ErrUnexpectedEOF
		return
	}

	tp = b[i]
	mut = b[i+1]
	i += 2

	return
}

func (d *LowDecoder) Section(b []byte, st int) (id byte, data []byte, i int, err error) {
	i = st

	if i+2 > len(b) {
		err = ErrUnexpectedEOF
		return
	}

	id = b[i]
	i++

	l, i, err := d.Int(b, i)
	if err != nil {
		return id, nil, st, err
	}

	if i+l > len(b) {
		return id, nil, st, ErrUnexpectedEOF
	}

	data = b[i : i+l]
	i += l

	return
}

func appendOrSet[B ~byte](copy bool, b []B, data ...B) []B {
	if !copy {
		return data
	}

	return append(b, data...)
}

func common(a, b []byte) (c int) {
	for c < len(a) && c < len(b) && a[c] == b[c] {
		c++
	}

	return
}

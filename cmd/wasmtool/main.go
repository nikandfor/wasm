package main

import (
	"io"
	"net"
	"net/http"
	"os"

	"nikand.dev/go/cli"
	"nikand.dev/go/cli/flag"
	"nikand.dev/go/wasm"
	"tlog.app/go/errors"
	"tlog.app/go/tlog"
	"tlog.app/go/tlog/ext/tlflag"
	"tlog.app/go/tlog/tlio"
	"tlog.app/go/tlog/tlwire"
)

type (
	bytearr []byte
)

func main() {
	dump := &cli.Command{
		Name:   "dump",
		Args:   cli.Args{},
		Action: dumpRun,
	}

	app := &cli.Command{
		Name:        "wasmtool",
		Description: "tool to work with wasm format",
		Before:      before,
		Flags: []*cli.Flag{
			cli.NewFlag("log", "stderr?dm", "log output file (or stderr)"),
			cli.NewFlag("verbosity,v", "", "logger verbosity topics"),
			cli.NewFlag("debug", "", "debug address", flag.Hidden),
			cli.FlagfileFlag,
			cli.HelpFlag,
		},
		Commands: []*cli.Command{
			dump,
		},
	}

	cli.RunAndExit(app, os.Args, os.Environ())
}

func before(c *cli.Command) error {
	w, err := tlflag.OpenWriter(c.String("log"))
	if err != nil {
		return errors.Wrap(err, "open log file")
	}

	err = tlio.WalkWriter(w, func(w io.Writer) error {
		c, ok := w.(*tlog.ConsoleWriter)
		if !ok {
			return nil
		}

		c.StringOnNewLineMinLen = 16

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "walk writer")
	}

	tlog.DefaultLogger = tlog.New(w)

	tlog.SetVerbosity(c.String("verbosity"))

	if q := c.String("debug"); q != "" {
		l, err := net.Listen("tcp", q)
		if err != nil {
			return errors.Wrap(err, "listen debug")
		}

		tlog.Printw("start debug interface", "addr", l.Addr())

		go func() {
			err := http.Serve(l, nil)
			if err != nil {
				tlog.Printw("debug", "addr", q, "err", err, "", tlog.Fatal)
				panic(err)
			}
		}()
	}

	return nil
}

func dumpRun(c *cli.Command) (err error) {
	var d wasm.Decoder

	for _, a := range c.Args {
		err := func() error {
			data, err := os.ReadFile(a)
			if err != nil {
				return errors.Wrap(err, "read file")
			}

			m := &wasm.Module{}

			err = d.Module(data, m)
			if err != nil {
				return errors.Wrap(err, "decode")
			}

			tlog.Printw("module", "start", m.Start, "sections", bytearr(m.Sections), "data_count", m.DataCount)

			for i, v := range m.Import {
				tlog.Printw("import", "i", i, "mod", v.Module, "name", v.Name)
			}

			for i, v := range m.Type {
				tlog.Printw("type", "i", i, "params", v.Params, "result", v.Result)
			}

			for i, v := range m.Function {
				tlog.Printw("function", "i", i, "tp", v)
			}

			for i, v := range m.Table {
				tlog.Printw("table", "i", i, "tp", v.Type, "limits", v.Limits)
			}

			for i, v := range m.Memory {
				tlog.Printw("memory", "i", i, "limits", v)
			}

			for i, v := range m.Global {
				tlog.Printw("global", "i", i, "tp", v.Type, "mut", v.Mut, "expr", v.Expr)
			}

			for i, v := range m.Export {
				tlog.Printw("export", "i", i, "name", v.Name, "exp_tp", v.ExportType, "index", v.Index)
			}

			for i, v := range m.Element {
				tlog.Printw("element", "i", i, "tp", v.Type, "expr", v.Expr, "funcs", v.Funcs)
			}

			for i, v := range m.Code {
				f, err := d.Func(v, wasm.FuncCode{})
				if err != nil {
					tlog.Printw("code", "i", i, "code", v, "err", err)
					continue
				}

				tlog.Printw("code", "i", i, "locals", f.Locals, "expr", f.Expr)
			}

			for i, v := range m.Data {
				tlog.Printw("data", "i", i, "expr", v.Expr, "init", v.Init)
			}

			for i, v := range m.Custom {
				tlog.Printw("custom", "i", i, "name", v.Name, "data", v.Data)
			}

			return nil
		}()
		if err != nil {
			return errors.Wrap(err, "%v", a)
		}
	}

	return nil
}

func (a bytearr) TlogAppend(b []byte) []byte {
	var e tlwire.Encoder

	b = e.AppendArray(b, len(a))

	for _, v := range a {
		b = e.AppendInt(b, int(v))
	}

	return b
}

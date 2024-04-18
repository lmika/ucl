package cmdlang

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type Inst struct {
	out io.Writer

	rootEC *evalCtx
}

type InstOption func(*Inst)

func WithOut(out io.Writer) InstOption {
	return func(i *Inst) {
		i.out = out
	}
}

func New(opts ...InstOption) *Inst {
	rootEC := &evalCtx{}
	rootEC.root = rootEC

	rootEC.addCmd("echo", invokableFunc(echoBuiltin))
	rootEC.addCmd("set", invokableFunc(setBuiltin))
	rootEC.addCmd("toUpper", invokableStreamFunc(toUpperBuiltin))
	//rootEC.addCmd("cat", invokableFunc(catBuiltin))
	rootEC.addCmd("call", invokableFunc(callBuiltin))

	rootEC.addCmd("map", invokableStreamFunc(mapBuiltin))

	rootEC.addCmd("cat", invokableFunc(concatBuiltin))

	rootEC.addMacro("if", macroFunc(ifBuiltin))
	rootEC.addMacro("foreach", macroFunc(foreachBuiltin))
	rootEC.addMacro("proc", macroFunc(procBuiltin))

	//rootEC.addCmd("testTimebomb", invokableStreamFunc(errorTestBuiltin))

	rootEC.setVar("hello", strObject("world"))

	inst := &Inst{
		out:    os.Stdout,
		rootEC: rootEC,
	}

	for _, opt := range opts {
		opt(inst)
	}

	return inst
}

func (inst *Inst) Out() io.Writer {
	if inst.out == nil {
		return os.Stdout
	}
	return inst.out
}

func (inst *Inst) Eval(ctx context.Context, expr string) (any, error) {
	res, err := inst.eval(ctx, expr)
	if err != nil {
		return nil, err
	}

	goRes, ok := toGoValue(res)
	if !ok {
		return nil, errors.New("result not convertable to go")
	}

	return goRes, nil
}

func (inst *Inst) eval(ctx context.Context, expr string) (object, error) {
	ast, err := parse(strings.NewReader(expr))
	if err != nil {
		return nil, err
	}

	eval := evaluator{inst: inst}

	return eval.evalScript(ctx, inst.rootEC, ast)
}

func (inst *Inst) EvalAndDisplay(ctx context.Context, expr string) error {
	res, err := inst.eval(ctx, expr)
	if err != nil {
		return err
	}

	return inst.display(ctx, res)
}

func (inst *Inst) display(ctx context.Context, res object) (err error) {
	switch v := res.(type) {
	case nil:
		if _, err = fmt.Fprintln(inst.out, "(nil)"); err != nil {
			return err
		}
	case stream:
		return forEach(v, func(o object, _ int) error { return inst.display(ctx, o) })
	default:
		if _, err = fmt.Fprintln(inst.out, v.String()); err != nil {
			return err
		}
	}
	return nil
}

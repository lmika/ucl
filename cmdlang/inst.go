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
	rootEC.addCmd("toUpper", invokableFunc(toUpperBuiltin))
	//rootEC.addCmd("cat", invokableFunc(catBuiltin))
	rootEC.addCmd("index", invokableFunc(indexBuiltin))
	rootEC.addCmd("call", invokableFunc(callBuiltin))

	rootEC.addCmd("map", invokableFunc(mapBuiltin))
	rootEC.addCmd("head", invokableFunc(firstBuiltin))

	rootEC.addCmd("eq", invokableFunc(eqBuiltin))
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

	// TODO: this should be a separate forkAndIsolate() session
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
	default:
		if _, err = fmt.Fprintln(inst.out, v.String()); err != nil {
			return err
		}
	}
	return nil
}

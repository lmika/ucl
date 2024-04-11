package cmdlang

import (
	"context"
	"fmt"
	"strings"
)

type Inst struct {
	rootEC *evalCtx
}

func New() *Inst {
	rootEC := evalCtx{}
	rootEC.addCmd("echo", invokableFunc(echoBuiltin))
	rootEC.addCmd("set", invokableFunc(setBuiltin))
	rootEC.addCmd("toUpper", invokableStreamFunc(toUpperBuiltin))
	rootEC.addCmd("cat", invokableFunc(catBuiltin))

	rootEC.addCmd("testTimebomb", invokableStreamFunc(errorTestBuiltin))

	rootEC.setVar("hello", strObject("world"))

	return &Inst{
		rootEC: &rootEC,
	}
}

// TODO: return value?
func (inst *Inst) Eval(ctx context.Context, expr string) (any, error) {
	ast, err := parse(strings.NewReader(expr))
	if err != nil {
		return nil, err
	}

	eval := evaluator{}
	return eval.evaluate(ctx, inst.rootEC, ast)
}

func (inst *Inst) EvalAndDisplay(ctx context.Context, expr string) error {
	res, err := inst.Eval(ctx, expr)
	if err != nil {
		return err
	}

	return inst.display(ctx, res)
}

func (inst *Inst) display(ctx context.Context, res object) (err error) {
	switch v := res.(type) {
	case stream:
		return forEach(v, func(o object) error { return inst.display(ctx, o) })
	case string:
		fmt.Println(v)
	}
	return nil
}

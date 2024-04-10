package cmdlang

import (
	"context"
	"strings"
)

type Inst struct {
	rootEC *evalCtx
}

func New() *Inst {
	rootEC := evalCtx{}
	rootEC.addCmd("echo", invokableFunc(echoBuiltin))

	return &Inst{
		rootEC: &rootEC,
	}
}

// TODO: return value?
func (inst *Inst) Eval(ctx context.Context, expr string) error {
	ast, err := parse(strings.NewReader(expr))
	if err != nil {
		return err
	}

	eval := evaluator{}
	return eval.evaluate(ctx, inst.rootEC, ast)
}

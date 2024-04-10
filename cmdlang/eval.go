package cmdlang

import (
	"context"
	"errors"
	"github.com/lmika/gopkgs/fp/slices"
	"strconv"
)

type evaluator struct {
}

func (e evaluator) evaluate(ctx context.Context, ec *evalCtx, ast *astCmd) error {
	cmd, err := ec.lookupCmd(ast.Name)
	if err != nil {
		return err
	}

	args, err := slices.MapWithError(ast.Args, func(a astCmdArg) (string, error) {
		return e.evalArg(ctx, ec, a)
	})
	if err != nil {
		return err
	}

	return cmd.invoke(ctx, invocationArgs{
		args: args,
	})
}

func (e evaluator) evalArg(ctx context.Context, ec *evalCtx, n astCmdArg) (string, error) {
	return e.evalLiteral(ctx, ec, n.Literal)
}

func (e evaluator) evalLiteral(ctx context.Context, ec *evalCtx, n astLiteral) (string, error) {
	switch {
	case n.Str != nil:
		uq, err := strconv.Unquote(*n.Str)
		if err != nil {
			return "", err
		}
		return uq, nil
	case n.Ident != nil:
		return *n.Ident, nil
	}
	return "", errors.New("unhandled literal type")
}

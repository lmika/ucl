package cmdlang

import (
	"context"
	"errors"
	"github.com/lmika/gopkgs/fp/slices"
	"strconv"
)

type evaluator struct {
}

func (e evaluator) evaluate(ctx context.Context, ec *evalCtx, n *astPipeline) (object, error) {
	res, err := e.evalCmd(ctx, ec, n.First)
	if err != nil {
		return nil, err
	}
	if len(n.Rest) == 0 {
		return res, nil
	}

	// Command is a pipeline, so build it out
	for _, rest := range n.Rest {
		out, err := e.evalCmd(ctx, ec.withCurrentStream(asStream(res)), rest)
		if err != nil {
			return nil, err
		}
		res = out
	}
	return res, nil
}

func (e evaluator) evalCmd(ctx context.Context, ec *evalCtx, ast *astCmd) (object, error) {
	cmd, err := ec.lookupCmd(ast.Name)
	if err != nil {
		return nil, err
	}

	args, err := slices.MapWithError(ast.Args, func(a astCmdArg) (string, error) {
		return e.evalArg(ctx, ec, a)
	})
	if err != nil {
		return nil, err
	}

	return cmd.invoke(ctx, invocationArgs{
		args:     args,
		inStream: ec.currentStream,
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

package cmdlang

import (
	"context"
	"errors"
	"fmt"
	"github.com/lmika/gopkgs/fp/slices"
	"strconv"
	"strings"
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

	args, err := slices.MapWithError(ast.Args, func(a astCmdArg) (object, error) {
		return e.evalArg(ctx, ec, a)
	})
	if err != nil {
		return nil, err
	}

	if ec.currentStream != nil {
		if si, ok := cmd.(streamInvokable); ok {
			return si.invokeWithStream(ctx, ec.currentStream, invocationArgs{args: args})
		} else {
			if err := ec.currentStream.close(); err != nil {
				return nil, err
			}
		}
	}

	return cmd.invoke(ctx, invocationArgs{args: args})
}

func (e evaluator) evalArg(ctx context.Context, ec *evalCtx, n astCmdArg) (object, error) {
	switch {
	case n.Literal != nil:
		return e.evalLiteral(ctx, ec, n.Literal)
	case n.Sub != nil:
		return e.evalSub(ctx, ec, n.Sub)
	}
	return nil, errors.New("unhandled arg type")
}

func (e evaluator) evalLiteral(ctx context.Context, ec *evalCtx, n *astLiteral) (object, error) {
	switch {
	case n.Str != nil:
		uq, err := strconv.Unquote(*n.Str)
		if err != nil {
			return "", err
		}
		return strObject(uq), nil
	case n.Ident != nil:
		return strObject(*n.Ident), nil
	}
	return "", errors.New("unhandled literal type")
}

func (e evaluator) evalSub(ctx context.Context, ec *evalCtx, n *astPipeline) (object, error) {
	pipelineRes, err := e.evaluate(ctx, ec, n)
	if err != nil {
		return "", err
	}

	switch v := pipelineRes.(type) {
	case stream:
		// TODO: use proper lists here, not a string join
		sb := strings.Builder{}
		if err := forEach(v, func(o object) error {
			// TODO: use o.String()
			sb.WriteString(fmt.Sprint(o))
			return nil
		}); err != nil {
			return "", err
		}

		return strObject(sb.String()), nil
	}
	return pipelineRes, nil
}

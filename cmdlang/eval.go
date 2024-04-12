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
	inst *Inst
}

func (e evaluator) evalStatement(ctx context.Context, ec *evalCtx, n *astStatements) (object, error) {
	res, err := e.evalPipeline(ctx, ec, n.First)
	if err != nil {
		return nil, err
	}
	if len(n.Rest) == 0 {
		return res, nil
	}

	for _, rest := range n.Rest {
		// Discard and close unused streams
		if s, isStream := res.(stream); isStream {
			if err := s.close(); err != nil {
				return nil, err
			}
		}

		out, err := e.evalPipeline(ctx, ec, rest)
		if err != nil {
			return nil, err
		}
		res = out
	}
	return res, nil
}

func (e evaluator) evalPipeline(ctx context.Context, ec *evalCtx, n *astPipeline) (object, error) {
	res, err := e.evalCmd(ctx, ec, nil, n.First)
	if err != nil {
		return nil, err
	}
	if len(n.Rest) == 0 {
		return res, nil
	}

	// Command is a pipeline, so build it out
	for _, rest := range n.Rest {
		out, err := e.evalCmd(ctx, ec, asStream(res), rest)
		if err != nil {
			return nil, err
		}
		res = out
	}
	return res, nil
}

func (e evaluator) evalCmd(ctx context.Context, ec *evalCtx, currentStream stream, ast *astCmd) (object, error) {
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

	invArgs := invocationArgs{ec: ec, inst: e.inst, args: args, currentStream: currentStream}

	if currentStream != nil {
		if si, ok := cmd.(streamInvokable); ok {
			return si.invokeWithStream(ctx, currentStream, invArgs)
		} else {
			if err := currentStream.close(); err != nil {
				return nil, err
			}
		}
	}

	return cmd.invoke(ctx, invArgs)
}

func (e evaluator) evalArg(ctx context.Context, ec *evalCtx, n astCmdArg) (object, error) {
	switch {
	case n.Literal != nil:
		return e.evalLiteral(ctx, ec, n.Literal)
	case n.Var != nil:
		v, ok := ec.getVar(*n.Var)
		if !ok {
			return nil, fmt.Errorf("unknown variable %s", *n.Var)
		}
		return v, nil
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
			return nil, err
		}
		return strObject(uq), nil
	case n.Ident != nil:
		return strObject(*n.Ident), nil
	}
	return nil, errors.New("unhandled literal type")
}

func (e evaluator) evalSub(ctx context.Context, ec *evalCtx, n *astPipeline) (object, error) {
	pipelineRes, err := e.evalPipeline(ctx, ec, n)
	if err != nil {
		return nil, err
	}

	switch v := pipelineRes.(type) {
	case stream:
		// TODO: use proper lists here, not a string join
		sb := strings.Builder{}
		if err := forEach(v, func(o object, _ int) error {
			// TODO: use o.String()
			sb.WriteString(fmt.Sprint(o))
			return nil
		}); err != nil {
			return nil, err
		}

		return strObject(sb.String()), nil
	}
	return pipelineRes, nil
}

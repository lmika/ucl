package cmdlang

import (
	"context"
	"errors"
	"strconv"
)

type evaluator struct {
	inst *Inst
}

func (e evaluator) evalBlock(ctx context.Context, ec *evalCtx, n *astBlock) (lastRes object, err error) {
	// TODO: push scope?

	for _, s := range n.Statements {
		lastRes, err = e.evalStatement(ctx, ec, s)
		if err != nil {
			return nil, err
		}
	}
	return lastRes, nil
}

func (e evaluator) evalScript(ctx context.Context, ec *evalCtx, n *astScript) (lastRes object, err error) {
	return e.evalStatement(ctx, ec, n.Statements)
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
	if cmd := ec.lookupInvokable(ast.Name); cmd != nil {
		return e.evalInvokable(ctx, ec, currentStream, ast, cmd)
	} else if macro := ec.lookupMacro(ast.Name); macro != nil {
		return e.evalMacro(ctx, ec, currentStream, ast, macro)
	}

	return nil, errors.New("unknown command")
}

func (e evaluator) evalInvokable(ctx context.Context, ec *evalCtx, currentStream stream, ast *astCmd, cmd invokable) (object, error) {
	var (
		pargs   listObject
		kwargs  map[string]*listObject
		argsPtr *listObject
	)

	argsPtr = &pargs
	for _, arg := range ast.Args {
		if ident := arg.Ident; ident != nil && (*ident)[0] == '-' {
			// Arg switch
			if kwargs == nil {
				kwargs = make(map[string]*listObject)
			}

			argsPtr = &listObject{}
			kwargs[(*ident)[1:]] = argsPtr
		} else {
			ae, err := e.evalArg(ctx, ec, arg)
			if err != nil {
				return nil, err
			}
			argsPtr.Append(ae)
		}
	}

	invArgs := invocationArgs{ec: ec, inst: e.inst, args: pargs, kwargs: kwargs, currentStream: currentStream}

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

func (e evaluator) evalMacro(ctx context.Context, ec *evalCtx, currentStream stream, ast *astCmd, cmd macroable) (object, error) {
	return cmd.invokeMacro(ctx, macroArgs{
		eval:          e,
		ec:            ec,
		currentStream: currentStream,
		ast:           ast,
	})
}

func (e evaluator) evalArg(ctx context.Context, ec *evalCtx, n astCmdArg) (object, error) {
	switch {
	case n.Literal != nil:
		return e.evalLiteral(ctx, ec, n.Literal)
	case n.Ident != nil:
		return strObject(*n.Ident), nil
	case n.Var != nil:
		if v, ok := ec.getVar(*n.Var); ok {
			return v, nil
		}
		return nil, nil
	case n.Sub != nil:
		return e.evalSub(ctx, ec, n.Sub)
	case n.ListOrHash != nil:
		return e.evalListOrHash(ctx, ec, n.ListOrHash)
	case n.Block != nil:
		return blockObject{block: n.Block}, nil
	}
	return nil, errors.New("unhandled arg type")
}

func (e evaluator) evalListOrHash(ctx context.Context, ec *evalCtx, loh *astListOrHash) (object, error) {
	if loh.EmptyList {
		return listObject{}, nil
	} else if loh.EmptyHash {
		return hashObject{}, nil
	}

	if firstIsHash := loh.Elements[0].Right != nil; firstIsHash {
		h := hashObject{}
		for _, el := range loh.Elements {
			if el.Right == nil {
				return nil, errors.New("miss-match of lists and hash")
			}

			n, err := e.evalArg(ctx, ec, el.Left)
			if err != nil {
				return nil, err
			}

			v, err := e.evalArg(ctx, ec, *el.Right)
			if err != nil {
				return nil, err
			}

			h[n.String()] = v
		}
		return h, nil
	}

	l := listObject{}
	for _, el := range loh.Elements {
		if el.Right != nil {
			return nil, errors.New("miss-match of lists and hash")
		}
		v, err := e.evalArg(ctx, ec, el.Left)
		if err != nil {
			return nil, err
		}
		l = append(l, v)
	}
	return l, nil
}

func (e evaluator) evalLiteral(ctx context.Context, ec *evalCtx, n *astLiteral) (object, error) {
	switch {
	case n.Str != nil:
		uq, err := strconv.Unquote(*n.Str)
		if err != nil {
			return nil, err
		}
		return strObject(uq), nil
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
		list := listObject{}
		if err := forEach(v, func(o object, _ int) error {
			list = append(list, o)
			return nil
		}); err != nil {
			return nil, err
		}

		return list, nil
	}
	return pipelineRes, nil
}

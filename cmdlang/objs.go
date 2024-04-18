package cmdlang

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

type object interface {
	String() string
	Truthy() bool
}

type listObject []object

func (lo *listObject) Append(o object) {
	*lo = append(*lo, o)
}

func (s listObject) String() string {
	return fmt.Sprintf("%v", []object(s))
}

func (s listObject) Truthy() bool {
	return len(s) > 0
}

type hashObject map[string]object

func (s hashObject) String() string {
	return fmt.Sprintf("%v", map[string]object(s))
}

func (s hashObject) Truthy() bool {
	return len(s) > 0
}

type strObject string

func (s strObject) String() string {
	return string(s)
}

func (s strObject) Truthy() bool {
	return string(s) != ""
}

func toGoValue(obj object) (interface{}, bool) {
	switch v := obj.(type) {
	case nil:
		return nil, true
	case strObject:
		return string(v), true
	case listObject:
		xs := make([]interface{}, 0, len(v))
		for _, va := range v {
			x, ok := toGoValue(va)
			if !ok {
				continue
			}
			xs = append(xs, x)
		}
		return xs, true
	case hashObject:
		xs := make(map[string]interface{})
		for k, va := range v {
			x, ok := toGoValue(va)
			if !ok {
				continue
			}
			xs[k] = x
		}
		return xs, true
	case proxyObject:
		return v.p, true
	}

	return nil, false
}

func fromGoValue(v any) (object, error) {
	switch t := v.(type) {
	case nil:
		return nil, nil
	case string:
		return strObject(t), nil
	default:
		return proxyObject{t}, nil
	}
}

type macroArgs struct {
	eval          evaluator
	ec            *evalCtx
	currentStream stream
	ast           *astCmd
	argShift      int
}

func (ma macroArgs) nargs() int {
	return len(ma.ast.Args[ma.argShift:])
}

func (ma *macroArgs) shift(n int) {
	ma.argShift += n
}

func (ma macroArgs) identIs(ctx context.Context, n int, expectedIdent string) bool {
	if n >= len(ma.ast.Args[ma.argShift:]) {
		return false
	}

	lit := ma.ast.Args[ma.argShift+n].Ident
	if lit == nil {
		return false
	}

	return *lit == expectedIdent
}

func (ma *macroArgs) shiftIdent(ctx context.Context) (string, bool) {
	if ma.argShift >= len(ma.ast.Args) {
		return "", false
	}

	lit := ma.ast.Args[ma.argShift].Ident
	if lit != nil {
		ma.argShift += 1
		return *lit, true
	}
	return "", false
}

func (ma macroArgs) evalArg(ctx context.Context, n int) (object, error) {
	if n >= len(ma.ast.Args[ma.argShift:]) {
		return nil, errors.New("not enough arguments") // FIX
	}

	return ma.eval.evalArg(ctx, ma.ec, ma.ast.Args[ma.argShift+n])
}

func (ma macroArgs) evalBlock(ctx context.Context, n int, args []object, pushScope bool) (object, error) {
	obj, err := ma.evalArg(ctx, n)
	if err != nil {
		return nil, err
	}

	block, ok := obj.(blockObject)
	if !ok {
		return nil, errors.New("not a block object")
	}

	ec := ma.ec
	if pushScope {
		ec = ec.fork()
	}
	for i, n := range block.block.Names {
		if i < len(args) {
			ec.setVar(n, args[i])
		}
	}

	return ma.eval.evalBlock(ctx, ec, block.block)
}

type invocationArgs struct {
	inst          *Inst
	ec            *evalCtx
	currentStream stream
	args          []object
	kwargs        map[string]*listObject
}

func (ia invocationArgs) expectArgn(x int) error {
	if len(ia.args) < x {
		return errors.New("expected at least " + strconv.Itoa(x) + " args")
	}
	return nil
}

func (ia invocationArgs) stringArg(i int) (string, error) {
	if len(ia.args) < i {
		return "", errors.New("expected at least " + strconv.Itoa(i) + " args")
	}
	s, ok := ia.args[i].(fmt.Stringer)
	if !ok {
		return "", errors.New("expected a string arg")
	}
	return s.String(), nil
}

func (ia invocationArgs) shift(i int) invocationArgs {
	return invocationArgs{
		inst:          ia.inst,
		ec:            ia.ec,
		currentStream: ia.currentStream,
		args:          ia.args[i:],
		kwargs:        ia.kwargs,
	}
}

// invokable is an object that can be executed as a command
type invokable interface {
	invoke(ctx context.Context, args invocationArgs) (object, error)
}

type macroable interface {
	invokeMacro(ctx context.Context, args macroArgs) (object, error)
}

type streamInvokable interface {
	invokable
	invokeWithStream(context.Context, stream, invocationArgs) (object, error)
}

type invokableFunc func(ctx context.Context, args invocationArgs) (object, error)

func (i invokableFunc) invoke(ctx context.Context, args invocationArgs) (object, error) {
	return i(ctx, args)
}

type invokableStreamFunc func(ctx context.Context, inStream stream, args invocationArgs) (object, error)

func (i invokableStreamFunc) invoke(ctx context.Context, args invocationArgs) (object, error) {
	return i(ctx, nil, args)
}

func (i invokableStreamFunc) invokeWithStream(ctx context.Context, inStream stream, args invocationArgs) (object, error) {
	return i(ctx, inStream, args)
}

type blockObject struct {
	block *astBlock
}

func (bo blockObject) String() string {
	return "block"
}

func (bo blockObject) Truthy() bool {
	return len(bo.block.Statements) > 0
}

type macroFunc func(ctx context.Context, args macroArgs) (object, error)

func (i macroFunc) invokeMacro(ctx context.Context, args macroArgs) (object, error) {
	return i(ctx, args)
}

func isTruthy(obj object) bool {
	if obj == nil {
		return false
	}
	return obj.Truthy()
}

type proxyObject struct {
	p interface{}
}

func (p proxyObject) String() string {
	return fmt.Sprintf("proxyObject{%T}", p.p)
}

func (p proxyObject) Truthy() bool {
	//TODO implement me
	panic("implement me")
}

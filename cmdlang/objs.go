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
	}

	return nil, false
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

func (ma macroArgs) evalArg(ctx context.Context, n int) (object, error) {
	if n >= len(ma.ast.Args[ma.argShift:]) {
		return nil, errors.New("not enough arguments") // FIX
	}

	return ma.eval.evalArg(ctx, ma.ec, ma.ast.Args[ma.argShift+n])
}

func (ma macroArgs) evalBlock(ctx context.Context, n int) (object, error) {
	obj, err := ma.evalArg(ctx, n)
	if err != nil {
		return nil, err
	}

	block, ok := obj.(blockObject)
	if !ok {
		return nil, errors.New("not a block object")
	}

	return ma.eval.evalBlock(ctx, ma.ec, block.block)
}

type invocationArgs struct {
	inst          *Inst
	ec            *evalCtx
	currentStream stream
	args          []object
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

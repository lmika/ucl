package cmdlang

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

type object interface {
	String() string
	Truthy() bool
}

type listable interface {
	Len() int
	Index(i int) object
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

func (s listObject) Len() int {
	return len(s)
}

func (s listObject) Index(i int) object {
	return s[i]
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

type boolObject bool

func (b boolObject) String() string {
	if b {
		return "(true)"
	}
	return "(false))"
}

func (b boolObject) Truthy() bool {
	return bool(b)
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
	case listableProxyObject:
		return v.v.Interface(), true
	}

	return nil, false
}

func fromGoValue(v any) (object, error) {
	switch t := v.(type) {
	case nil:
		return nil, nil
	case string:
		return strObject(t), nil
	}

	resVal := reflect.ValueOf(v)
	if resVal.Type().Kind() == reflect.Slice {
		return listableProxyObject{resVal}, nil
	}

	return proxyObject{v}, nil
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

// streamableSource takes a stream.  If the stream is set, the inStream and invocation arguments are consumed as is.
// If not, then the first argument is consumed and returned as a stream.
func (ia invocationArgs) streamableSource(inStream stream) (invocationArgs, stream, error) {
	if inStream != nil {
		return ia, inStream, nil
	}

	if len(ia.args) < 1 {
		return ia, nil, errors.New("expected at least 1 argument")
	}

	switch v := ia.args[0].(type) {
	case listObject:
		return ia.shift(1), &listIterStream{list: v}, nil
	}

	return ia, nil, errors.New("expected arg 0 to be streamable")
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

func (ia invocationArgs) fork(currentStr stream, args []object) invocationArgs {
	return invocationArgs{
		inst:          ia.inst,
		ec:            ia.ec,
		currentStream: currentStr,
		args:          args,
		kwargs:        make(map[string]*listObject),
	}
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

type listableProxyObject struct {
	v reflect.Value
}

func (p listableProxyObject) String() string {
	return fmt.Sprintf("listableProxyObject{%v}", p.v.Type())
}

func (p listableProxyObject) Truthy() bool {
	panic("implement me")
}

func (p listableProxyObject) Len() int {
	return p.v.Len()
}

func (p listableProxyObject) Index(i int) object {
	e, err := fromGoValue(p.v.Index(i).Interface())
	if err != nil {
		return nil
	}
	return e
}

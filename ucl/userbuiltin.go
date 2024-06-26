package ucl

import (
	"context"
	"errors"
	"reflect"

	"github.com/lmika/gopkgs/fp/slices"
)

type BuiltinHandler func(ctx context.Context, args CallArgs) (any, error)

type MissingBuiltinHandler func(ctx context.Context, name string, args CallArgs) (any, error)

type CallArgs struct {
	args invocationArgs
}

func (ca *CallArgs) NArgs() int {
	return len(ca.args.args)
}

func (ca *CallArgs) Bind(vars ...interface{}) error {
	if len(ca.args.args) < len(vars) {
		return errors.New("wrong number of arguments")
	}

	for i, v := range vars {
		if err := ca.bindArg(v, ca.args.args[i]); err != nil {
			return err
		}
	}
	ca.args = ca.args.shift(len(vars))
	return nil
}

func (ca *CallArgs) CanBind(vars ...interface{}) bool {
	if len(ca.args.args) < len(vars) {
		return false
	}

	for i, v := range vars {
		if !canBindArg(v, ca.args.args[i]) {
			return false
		}
	}
	return true
}

func (ca *CallArgs) Shift(n int) {
	ca.args = ca.args.shift(n)
}

func (ca CallArgs) IsTopLevel() bool {
	return ca.args.ec.parent == nil || ca.args.ec == ca.args.ec.root
}

func (ca CallArgs) HasSwitch(name string) bool {
	if ca.args.kwargs == nil {
		return false
	}

	_, ok := ca.args.kwargs[name]
	return ok
}

func (ca CallArgs) BindSwitch(name string, val interface{}) error {
	if ca.args.kwargs == nil {
		return nil
	}

	vars, ok := ca.args.kwargs[name]
	if !ok || len(*vars) != 1 {
		return nil
	}

	return ca.bindArg(val, (*vars)[0])
}

func (inst *Inst) SetBuiltin(name string, fn BuiltinHandler) {
	inst.rootEC.addCmd(name, userBuiltin{fn: fn})
}

type userBuiltin struct {
	fn func(ctx context.Context, args CallArgs) (any, error)
}

func (u userBuiltin) invoke(ctx context.Context, args invocationArgs) (object, error) {
	v, err := u.fn(ctx, CallArgs{args: args})
	if err != nil {
		return nil, err
	}

	return fromGoValue(v)
}

func (ca CallArgs) bindArg(v interface{}, arg object) error {
	switch t := v.(type) {
	case *interface{}:
		*t, _ = toGoValue(arg)
	case *Invokable:
		i, ok := arg.(invokable)
		if !ok {
			return errors.New("exepected invokable")
		}
		*t = Invokable{
			inv:  i,
			eval: ca.args.eval,
			inst: ca.args.inst,
			ec:   ca.args.ec,
		}
		return nil
	case *string:
		*t = arg.String()
	case *int:
		if iArg, ok := arg.(intObject); ok {
			*t = int(iArg)
		} else {
			return errors.New("invalid arg")
		}
	}

	switch t := arg.(type) {
	case OpaqueObject:
		if v == nil {
			return errors.New("opaque object not bindable to nil")
		}

		vr := reflect.ValueOf(v)
		tr := reflect.ValueOf(t.v)
		if vr.Kind() != reflect.Pointer {
			return errors.New("expected pointer for an opaque object bind")
		}

		if !tr.Type().AssignableTo(vr.Elem().Type()) {
			return errors.New("opaque object not assignable to passed in value")
		}

		vr.Elem().Set(tr)
		return nil
	case proxyObject:
		return bindProxyObject(v, reflect.ValueOf(t.p))
	case listableProxyObject:
		return bindProxyObject(v, t.v)
	case structProxyObject:
		return bindProxyObject(v, t.v)
	}

	return nil
}

func canBindArg(v interface{}, arg object) bool {
	switch v.(type) {
	case *string:
		return true
	case *int:
		_, ok := arg.(intObject)
		return ok
	}

	switch t := arg.(type) {
	case OpaqueObject:
		vr := reflect.ValueOf(v)
		tr := reflect.ValueOf(t.v)
		if vr.Kind() != reflect.Pointer {
			return false
		}

		if !tr.Type().AssignableTo(vr.Elem().Type()) {
			return false
		}

		return true
	case proxyObject:
		return canBindProxyObject(v, reflect.ValueOf(t.p))
	case listableProxyObject:
		return canBindProxyObject(v, t.v)
	case structProxyObject:
		return canBindProxyObject(v, t.v)
	}

	return true
}

func bindProxyObject(v interface{}, r reflect.Value) error {
	argValue := reflect.ValueOf(v)
	if argValue.Kind() != reflect.Ptr {
		return errors.New("v must be a pointer to a struct")
	}

	for {
		if r.Type().AssignableTo(argValue.Elem().Type()) {
			argValue.Elem().Set(r)
			return nil
		}
		if r.Type().Kind() != reflect.Pointer {
			return nil
		}

		r = r.Elem()
	}
}

func canBindProxyObject(v interface{}, r reflect.Value) bool {
	argValue := reflect.ValueOf(v)
	if argValue.Kind() != reflect.Ptr {
		return false
	}

	for {
		if r.Type().AssignableTo(argValue.Elem().Type()) {
			argValue.Elem().Set(r)
			return true
		}
		if r.Type().Kind() != reflect.Pointer {
			return true
		}

		r = r.Elem()
	}
}

func (inst *Inst) missingHandlerInvokable(name string) missingHandlerInvokable {
	return missingHandlerInvokable{name: name, handler: inst.missingBuiltinHandler}
}

type missingHandlerInvokable struct {
	name    string
	handler MissingBuiltinHandler
}

func (m missingHandlerInvokable) invoke(ctx context.Context, args invocationArgs) (object, error) {
	v, err := m.handler(ctx, m.name, CallArgs{args: args})
	if err != nil {
		return nil, err
	}

	return fromGoValue(v)
}

type Invokable struct {
	inv  invokable
	eval evaluator
	inst *Inst
	ec   *evalCtx
}

func (i Invokable) IsNil() bool {
	return i.inv == nil
}

func (i Invokable) Invoke(ctx context.Context, args ...any) (any, error) {
	if i.inv == nil {
		return nil, nil
	}

	var err error
	invArgs := invocationArgs{
		eval: i.eval,
		ec:   i.ec,
		inst: i.inst,
	}

	invArgs.args, err = slices.MapWithError(args, func(a any) (object, error) {
		return fromGoValue(a)
	})
	if err != nil {
		return nil, err
	}

	res, err := i.inv.invoke(ctx, invArgs)
	if err != nil {
		return nil, err
	}

	goRes, ok := toGoValue(res)
	if !ok {
		return nil, errors.New("cannot convert result to Go Value")
	}
	return goRes, err
}

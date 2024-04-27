package ucl

import (
	"context"
	"errors"
	"reflect"
)

type CallArgs struct {
	args invocationArgs
}

func (ca *CallArgs) Bind(vars ...interface{}) error {
	if len(ca.args.args) < len(vars) {
		return errors.New("wrong number of arguments")
	}

	for i, v := range vars {
		if err := bindArg(v, ca.args.args[i]); err != nil {
			return err
		}
	}
	ca.args = ca.args.shift(len(vars))
	return nil
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

	return bindArg(val, (*vars)[0])
}

func (inst *Inst) SetBuiltin(name string, fn func(ctx context.Context, args CallArgs) (any, error)) {
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

func bindArg(v interface{}, arg object) error {
	switch t := v.(type) {
	case *string:
		*t = arg.String()
	}

	switch t := arg.(type) {
	case proxyObject:
		return bindProxyObject(v, reflect.ValueOf(t.p))
	case listableProxyObject:
		return bindProxyObject(v, t.v)
	case structProxyObject:
		return bindProxyObject(v, t.v)
	}

	return nil
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

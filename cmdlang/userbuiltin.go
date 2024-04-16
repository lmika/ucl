package cmdlang

import (
	"context"
	"errors"
	"reflect"
)

type CallArgs struct {
	args invocationArgs
}

func (ca CallArgs) Bind(vars ...interface{}) error {
	if len(ca.args.args) != len(vars) {
		return errors.New("wrong number of arguments")
	}

	for i, v := range vars {
		switch t := v.(type) {
		case *string:
			tv, err := ca.args.stringArg(i)
			if err != nil {
				return err
			}
			*t = tv
		}

		// Check for proxy object
		if po, ok := ca.args.args[i].(proxyObject); ok {
			poValue := reflect.ValueOf(po.p)
			argValue := reflect.ValueOf(v)

			if argValue.Type().Kind() != reflect.Pointer {
				continue
			} else if !poValue.Type().AssignableTo(argValue.Elem().Type()) {
				continue
			}

			argValue.Elem().Set(poValue)
		}
	}
	return nil
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

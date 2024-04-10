package cmdlang

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

type object interface {
}

type strObject string

func (s strObject) String() string {
	return string(s)
}

type invocationArgs struct {
	args     []object
	inStream stream
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

type invokableFunc func(ctx context.Context, args invocationArgs) (object, error)

func (i invokableFunc) invoke(ctx context.Context, args invocationArgs) (object, error) {
	return i(ctx, args)
}

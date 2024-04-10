package cmdlang

import (
	"context"
	"errors"
	"strconv"
)

type object = any

type invocationArgs struct {
	args     []string
	inStream stream
}

func (ia invocationArgs) expectArgn(x int) error {
	if len(ia.args) < x {
		return errors.New("expected at least " + strconv.Itoa(x) + " args")
	}
	return nil
}

// invokable is an object that can be executed as a command
type invokable interface {
	invoke(ctx context.Context, args invocationArgs) (object, error)
}

type invokableFunc func(ctx context.Context, args invocationArgs) (object, error)

func (i invokableFunc) invoke(ctx context.Context, args invocationArgs) (object, error) {
	return i(ctx, args)
}

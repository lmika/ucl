package cmdlang

import "context"

type invocationArgs struct {
	args []string
}

type invokable interface {
	invoke(ctx context.Context, args invocationArgs) error
}

type invokableFunc func(ctx context.Context, args invocationArgs) error

func (i invokableFunc) invoke(ctx context.Context, args invocationArgs) error {
	return i(ctx, args)
}

package cmdlang_test

import (
	"context"
	"testing"

	"github.com/lmika/cmdlang-proto/cmdlang"
	"github.com/stretchr/testify/assert"
)

func TestInst_SetBuiltin(t *testing.T) {
	t.Run("simple builtin accepting and returning strings", func(t *testing.T) {
		inst := cmdlang.New()
		inst.SetBuiltin("add2", func(ctx context.Context, args cmdlang.CallArgs) (any, error) {
			var x, y string

			if err := args.Bind(&x, &y); err != nil {
				return nil, err
			}

			return x + y, nil
		})

		res, err := inst.Eval(context.Background(), `add2 "Hello, " "World"`)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, World", res)
	})

	t.Run("builtin return proxy object", func(t *testing.T) {
		type pair struct {
			x, y string
		}

		inst := cmdlang.New()
		inst.SetBuiltin("add2", func(ctx context.Context, args cmdlang.CallArgs) (any, error) {
			var x, y string

			if err := args.Bind(&x, &y); err != nil {
				return nil, err
			}

			return pair{x, y}, nil
		})

		res, err := inst.Eval(context.Background(), `add2 "Hello" "World"`)
		assert.NoError(t, err)
		assert.Equal(t, pair{"Hello", "World"}, res)
	})

	t.Run("builtin operating on and returning proxy object", func(t *testing.T) {
		type pair struct {
			x, y string
		}

		tests := []struct {
			descr string
			expr  string
			want  string
		}{
			{descr: "pass via args", expr: `join (add2 "left" "right")`, want: "left:right"},
			{descr: "pass via vars", expr: `set x (add2 "blue" "green") ; join $x`, want: "blue:green"},
		}

		for _, tt := range tests {
			t.Run(tt.descr, func(t *testing.T) {
				inst := cmdlang.New()
				inst.SetBuiltin("add2", func(ctx context.Context, args cmdlang.CallArgs) (any, error) {
					var x, y string

					if err := args.Bind(&x, &y); err != nil {
						return nil, err
					}

					return pair{x, y}, nil
				})
				inst.SetBuiltin("join", func(ctx context.Context, args cmdlang.CallArgs) (any, error) {
					var x pair

					if err := args.Bind(&x); err != nil {
						return nil, err
					}

					return x.x + ":" + x.y, nil
				})

				res, err := inst.Eval(context.Background(), tt.expr)
				assert.NoError(t, err)
				assert.Equal(t, tt.want, res)
			})
		}
	})
}

func TestCallArgs_Bind(t *testing.T) {
	t.Run("bind to an interface", func(t *testing.T) {
		ctx := context.Background()

		inst := cmdlang.New()
		inst.SetBuiltin("sa", func(ctx context.Context, args cmdlang.CallArgs) (any, error) {
			return doStringA{this: "a val"}, nil
		})
		inst.SetBuiltin("sb", func(ctx context.Context, args cmdlang.CallArgs) (any, error) {
			return doStringB{left: "foo", right: "bar"}, nil
		})
		inst.SetBuiltin("dostr", func(ctx context.Context, args cmdlang.CallArgs) (any, error) {
			var ds doStringable

			if err := args.Bind(&ds); err != nil {
				return nil, err
			}

			return ds.DoString(), nil
		})

		va, err := inst.Eval(ctx, `dostr (sa)`)
		assert.NoError(t, err)
		assert.Equal(t, "do string A: a val", va)

		vb, err := inst.Eval(ctx, `dostr (sb)`)
		assert.NoError(t, err)
		assert.Equal(t, "do string B: foo bar", vb)
	})
}

type doStringable interface {
	DoString() string
}

type doStringA struct {
	this string
}

func (da doStringA) DoString() string {
	return "do string A: " + da.this
}

type doStringB struct {
	left, right string
}

func (da doStringB) DoString() string {
	return "do string B: " + da.left + " " + da.right
}

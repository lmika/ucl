package cmdlang_test

import (
	"context"
	"github.com/lmika/cmdlang-proto/cmdlang"
	"github.com/stretchr/testify/assert"
	"testing"
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

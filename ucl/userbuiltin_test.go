package ucl_test

import (
	"bytes"
	"context"
	"github.com/lmika/ucl/ucl"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInst_SetBuiltin(t *testing.T) {
	t.Run("simple builtin accepting and returning strings", func(t *testing.T) {
		inst := ucl.New()
		inst.SetBuiltin("add2", func(ctx context.Context, args ucl.CallArgs) (any, error) {
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

	t.Run("bind shift arguments", func(t *testing.T) {
		inst := ucl.New()
		inst.SetBuiltin("add2", func(ctx context.Context, args ucl.CallArgs) (any, error) {
			var x, y string

			if err := args.Bind(&x); err != nil {
				return nil, err
			}
			if err := args.Bind(&y); err != nil {
				return nil, err
			}

			return x + y, nil
		})

		res, err := inst.Eval(context.Background(), `add2 "Hello, " "World"`)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, World", res)
	})

	t.Run("simple builtin with optional switches and strings", func(t *testing.T) {
		inst := ucl.New()
		inst.SetBuiltin("add2", func(ctx context.Context, args ucl.CallArgs) (any, error) {
			var x, y, sep string

			if err := args.BindSwitch("sep", &sep); err != nil {
				return nil, err
			}
			if err := args.BindSwitch("left", &x); err != nil {
				return nil, err
			}
			if err := args.BindSwitch("right", &y); err != nil {
				return nil, err
			}

			v := x + sep + y
			if args.HasSwitch("upcase") {
				v = strings.ToUpper(v)
			}

			return v, nil
		})

		tests := []struct {
			descr string
			expr  string
			want  string
		}{
			{"plain eval", `add2 -sep ", " -right "world" -left "Hello"`, "Hello, world"},
			{"swap 1", `add2 -right "right" -left "left" -sep ":"`, "left:right"},
			{"swap 2", `add2 -left "left" -sep ":" -right "right" -upcase`, "LEFT:RIGHT"},
		}

		for _, tt := range tests {
			t.Run(tt.descr, func(t *testing.T) {
				res, err := inst.Eval(context.Background(), tt.expr)
				assert.NoError(t, err)
				assert.Equal(t, tt.want, res)
			})
		}
	})

	t.Run("builtin return proxy object", func(t *testing.T) {
		type pair struct {
			x, y string
		}

		inst := ucl.New()
		inst.SetBuiltin("add2", func(ctx context.Context, args ucl.CallArgs) (any, error) {
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
				inst := ucl.New()
				inst.SetBuiltin("add2", func(ctx context.Context, args ucl.CallArgs) (any, error) {
					var x, y string

					if err := args.Bind(&x, &y); err != nil {
						return nil, err
					}

					return pair{x, y}, nil
				})
				inst.SetBuiltin("join", func(ctx context.Context, args ucl.CallArgs) (any, error) {
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

	t.Run("slices returned by commands treated as lists", func(t *testing.T) {
		tests := []struct {
			descr   string
			expr    string
			want    any
			wantOut string
		}{
			{descr: "return as is", expr: `countTo3`, want: []string{"1", "2", "3"}},
			{descr: "iterate over", expr: `foreach (countTo3) { |x| echo $x }`, wantOut: "1\n2\n3\n"},
		}

		for _, tt := range tests {
			t.Run(tt.descr, func(t *testing.T) {
				outW := bytes.NewBuffer(nil)
				inst := ucl.New(ucl.WithOut(outW))

				inst.SetBuiltin("countTo3", func(ctx context.Context, args ucl.CallArgs) (any, error) {
					return []string{"1", "2", "3"}, nil
				})

				res, err := inst.Eval(context.Background(), tt.expr)
				assert.NoError(t, err)
				assert.Equal(t, tt.want, res)
				assert.Equal(t, tt.wantOut, outW.String())
			})
		}
	})
}

func TestCallArgs_Bind(t *testing.T) {
	t.Run("bind to an interface", func(t *testing.T) {
		ctx := context.Background()

		inst := ucl.New()
		inst.SetBuiltin("sa", func(ctx context.Context, args ucl.CallArgs) (any, error) {
			return doStringA{this: "a val"}, nil
		})
		inst.SetBuiltin("sb", func(ctx context.Context, args ucl.CallArgs) (any, error) {
			return doStringB{left: "foo", right: "bar"}, nil
		})
		inst.SetBuiltin("dostr", func(ctx context.Context, args ucl.CallArgs) (any, error) {
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

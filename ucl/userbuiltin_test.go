package ucl_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"ucl.lmika.dev/ucl"

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

	t.Run("builtin return proxy object ptr", func(t *testing.T) {
		type pair struct {
			x, y string
		}

		inst := ucl.New()
		inst.SetBuiltin("add2", func(ctx context.Context, args ucl.CallArgs) (any, error) {
			var x, y string

			if err := args.Bind(&x, &y); err != nil {
				return nil, err
			}

			return &pair{x, y}, nil
		})

		res, err := inst.Eval(context.Background(), `add2 "Hello" "World"`)
		assert.NoError(t, err)
		assert.Equal(t, &pair{"Hello", "World"}, res)
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

	t.Run("opaques returned as is", func(t *testing.T) {
		type opaqueThingType struct {
			x string
			y string
			z string
		}
		opaqueThing := &opaqueThingType{x: "do", y: "not", z: "touch"}

		tests := []struct {
			descr   string
			expr    string
			wantErr bool
		}{
			{descr: "return as is", expr: `getOpaque`, wantErr: false},
			{descr: "carry around ok", expr: `set x (getOpaque) ; $x`, wantErr: false},
			{descr: "iterate over", expr: `foreach (countTo3) { |x| echo $x }`, wantErr: true},
		}

		for _, tt := range tests {
			t.Run(tt.descr, func(t *testing.T) {

				outW := bytes.NewBuffer(nil)
				inst := ucl.New(ucl.WithOut(outW))

				inst.SetBuiltin("getOpaque", func(ctx context.Context, args ucl.CallArgs) (any, error) {
					return ucl.Opaque(opaqueThing), nil
				})

				res, err := inst.Eval(context.Background(), tt.expr)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Same(t, opaqueThing, res)
				}
			})
		}
	})

	t.Run("operate on opaques", func(t *testing.T) {
		type opaqueThingType struct {
			x string
			y string
			z string
		}
		opaqueThing := &opaqueThingType{x: "do", y: "not", z: "touch"}

		tests := []struct {
			descr   string
			expr    string
			want    opaqueThingType
			wantErr bool
		}{
			{descr: "return as is", expr: `getOpaque`, want: *opaqueThing},
			{descr: "update pointer 1", expr: `set x (getOpaque) ; setProp $x -x "do" -y "touch" -z "this"`, want: opaqueThingType{x: "do", y: "touch", z: "this"}},
			{descr: "update pointer 2", expr: `set x (getOpaque) ; setProp $x -x "yes" ; setProp $x -y "this" -z "too"`, want: opaqueThingType{x: "yes", y: "this", z: "too"}},
			{descr: "bad args", expr: `set x (getOpaque) ; setProp $t -x "yes" ; setProp $bla -y "this" -z "too"`, want: *opaqueThing, wantErr: true},
		}

		for _, tt := range tests {
			t.Run(tt.descr, func(t *testing.T) {

				outW := bytes.NewBuffer(nil)
				inst := ucl.New(ucl.WithOut(outW))

				inst.SetBuiltin("getOpaque", func(ctx context.Context, args ucl.CallArgs) (any, error) {
					return ucl.Opaque(opaqueThing), nil
				})
				inst.SetBuiltin("setProp", func(ctx context.Context, args ucl.CallArgs) (any, error) {
					var o *opaqueThingType

					if err := args.Bind(&o); err != nil {
						return nil, err
					} else if o == nil {
						return nil, errors.New("is nil")
					}

					if args.HasSwitch("x") {
						var s string
						_ = args.BindSwitch("x", &s)
						o.x = s
					}
					if args.HasSwitch("y") {
						var s string
						_ = args.BindSwitch("y", &s)
						o.y = s
					}
					if args.HasSwitch("z") {
						var s string
						_ = args.BindSwitch("z", &s)
						o.z = s
					}

					return nil, nil
				})

				_, err := inst.Eval(context.Background(), tt.expr)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.want, *opaqueThing)
				}
			})
		}
	})
}

func TestCallArgs_Bind(t *testing.T) {
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
}

func TestCallArgs_CanBind(t *testing.T) {
	tests := []struct {
		descr string
		eval  string
		want  []string
	}{
		{descr: "bind nothing", eval: `test`, want: []string{}},
		{descr: "bind one", eval: `test "yes"`, want: []string{"str"}},
		{descr: "bind two", eval: `test "yes" 213`, want: []string{"str", "int"}},
		{descr: "bind three", eval: `test "yes" 213 (proxy)`, want: []string{"all", "str", "int", "proxy"}},
	}

	for _, tt := range tests {
		t.Run(tt.descr, func(t *testing.T) {
			type proxyObj struct{}

			ctx := context.Background()
			res := make([]string, 0)

			inst := ucl.New()
			inst.SetBuiltin("proxy", func(ctx context.Context, args ucl.CallArgs) (any, error) {
				return proxyObj{}, nil
			})
			inst.SetBuiltin("test", func(ctx context.Context, args ucl.CallArgs) (any, error) {
				var (
					s string
					i int
					p proxyObj
				)

				if args.CanBind(&s, &i, &p) {
					res = append(res, "all")
				}
				if args.CanBind(&s) {
					res = append(res, "str")
				}
				args.Shift(1)
				if args.CanBind(&i) {
					res = append(res, "int")
				}
				args.Shift(1)
				if args.CanBind(&p) {
					res = append(res, "proxy")
				}
				return nil, nil
			})

			_, err := inst.Eval(ctx, tt.eval)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}

	t.Run("can bind invokable", func(t *testing.T) {
		inst := ucl.New()
		inst.SetBuiltin("wrap", func(ctx context.Context, args ucl.CallArgs) (any, error) {
			var inv ucl.Invokable

			if err := args.Bind(&inv); err != nil {
				return nil, err
			}

			res, err := inv.Invoke(ctx, "hello")
			if err != nil {
				return nil, err
			}

			return fmt.Sprintf("[[%v]]", res), nil
		})

		ctx := context.Background()

		res, err := inst.Eval(ctx, `wrap { |x| toUpper $x }`)
		assert.NoError(t, err)
		assert.Equal(t, "[[HELLO]]", res)
	})

	t.Run("can carry invokable outside of context", func(t *testing.T) {
		inst := ucl.New()
		var inv ucl.Invokable

		inst.SetBuiltin("wrap", func(ctx context.Context, args ucl.CallArgs) (any, error) {
			if err := args.Bind(&inv); err != nil {
				return nil, err
			}

			return nil, nil
		})

		ctx := context.Background()

		assert.True(t, inv.IsNil())

		before, err := inv.Invoke(ctx, "hello")
		assert.NoError(t, err)
		assert.Nil(t, before)

		res, err := inst.Eval(ctx, `wrap { |x| toUpper $x }`)
		assert.NoError(t, err)
		assert.Nil(t, res)

		assert.False(t, inv.IsNil())

		after, err := inv.Invoke(ctx, "hello")
		assert.NoError(t, err)
		assert.Equal(t, "HELLO", after)
	})
}

func TestCallArgs_MissingCommandHandler(t *testing.T) {
	tests := []struct {
		descr string
		eval  string
		want  string
	}{
		{descr: "alpha", eval: `alpha`, want: "was alpha"},
		{descr: "bravo", eval: `bravo "this"`, want: "was bravo: this"},
		{descr: "charlie", eval: `charlie`, want: "was charlie"},
	}

	for _, tt := range tests {
		t.Run(tt.descr, func(t *testing.T) {
			ctx := context.Background()

			inst := ucl.New(
				ucl.WithMissingBuiltinHandler(func(ctx context.Context, name string, args ucl.CallArgs) (any, error) {
					var msg string
					if err := args.Bind(&msg); err == nil {
						return fmt.Sprintf("was %v: %v", name, msg), nil
					}
					return fmt.Sprintf("was %v", name), nil
				}))

			res, err := inst.Eval(ctx, tt.eval)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

func TestCallArgs_IsTopLevel(t *testing.T) {
	t.Run("true if the command is running at the top-level frame", func(t *testing.T) {
		ctx := context.Background()
		res := make(map[string]bool)

		inst := ucl.New()
		inst.SetBuiltin("lvl", func(ctx context.Context, args ucl.CallArgs) (any, error) {
			var n string
			if err := args.Bind(&n); err != nil {
				return nil, err
			}

			res[n] = args.IsTopLevel()
			return nil, nil
		})

		_, err := inst.Eval(ctx, `lvl "one"`)
		assert.NoError(t, err)
		assert.True(t, res["one"])

		_, err = inst.Eval(ctx, `echo (lvl "two")`)
		assert.NoError(t, err)
		assert.True(t, res["two"])

		_, err = inst.Eval(ctx, `proc doLvl { |n| lvl $n } ; doLvl "three"`)
		assert.NoError(t, err)
		assert.False(t, res["three"])

		_, err = inst.Eval(ctx, `doLvl "four"`)
		assert.NoError(t, err)
		assert.False(t, res["four"])

		_, err = inst.Eval(ctx, `["a"] | map { |x| doLvl "five" ; $x }`)
		assert.NoError(t, err)
		assert.False(t, res["five"])

		_, err = inst.Eval(ctx, `if 1 { lvl "six" }`)
		assert.NoError(t, err)
		assert.True(t, res["six"])
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

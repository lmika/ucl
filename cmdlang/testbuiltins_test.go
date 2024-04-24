package cmdlang

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

// Builtins used for test
func WithTestBuiltin() InstOption {
	return func(i *Inst) {
		i.rootEC.addCmd("firstarg", invokableFunc(func(ctx context.Context, args invocationArgs) (object, error) {
			return args.args[0], nil
		}))

		i.rootEC.addCmd("sjoin", invokableFunc(func(ctx context.Context, args invocationArgs) (object, error) {
			if len(args.args) == 0 {
				return strObject(""), nil
			}

			var line strings.Builder
			for _, arg := range args.args {
				if s, ok := arg.(fmt.Stringer); ok {
					line.WriteString(s.String())
				}
			}

			return strObject(line.String()), nil
		}))

		i.rootEC.addCmd("list", invokableFunc(func(ctx context.Context, args invocationArgs) (object, error) {
			return listObject(args.args), nil
		}))

		i.rootEC.addCmd("joinpipe", invokableFunc(func(ctx context.Context, args invocationArgs) (object, error) {
			sb := strings.Builder{}

			lst, ok := args.args[0].(listable)
			if !ok {
				return strObject(""), nil
			}

			l := lst.Len()
			for x := 0; x < l; x++ {
				if x > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(lst.Index(x).String())
			}
			return strObject(sb.String()), nil
		}))

		i.rootEC.setVar("a", strObject("alpha"))
		i.rootEC.setVar("bee", strObject("buzz"))
	}
}

func TestBuiltins_Echo(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want string
	}{
		{desc: "no args", expr: `echo`, want: "\n"},
		{desc: "single arg", expr: `echo "hello"`, want: "hello\n"},
		{desc: "dual args", expr: `echo "hello " "world"`, want: "hello world\n"},
		{desc: "multi-line 1", expr: `
			echo "Hello"
			echo "world"
		`, want: "Hello\nworld\n"},
		{desc: "multi-line 2", expr: `
			echo "Hello"


			echo "world"
		`, want: "Hello\nworld\n"},
		{desc: "multi-line 3", expr: `

;;;
			echo "Hello"
;

			echo "world"
;
		`, want: "Hello\nworld\n"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())
			res, err := inst.Eval(ctx, tt.expr)

			assert.NoError(t, err)
			assert.Nil(t, res)
			assert.Equal(t, tt.want, outW.String())
		})
	}
}

func TestBuiltins_If(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want string
	}{
		{desc: "single then", expr: `
			set x "Hello"
			if $x {
				echo "true"
			}`, want: "true\n(nil)\n"},
		{desc: "single then and else", expr: `
			set x "Hello"
			if $x {
				echo "true"
			} else {
				echo "false"
			}`, want: "true\n(nil)\n"},
		{desc: "single then, elif and else", expr: `
			set x "Hello"
			if $y {
				echo "y is true"
			} elif $x {
				echo "x is true"
			} else {
				echo "nothings x"
			}`, want: "x is true\n(nil)\n"},
		{desc: "single then and elif, no else", expr: `
			set x "Hello"
			if $y {
				echo "y is true"
			} elif $x {
				echo "x is true"
			}`, want: "x is true\n(nil)\n"},
		{desc: "single then, two elif, and else", expr: `
			set x "Hello"
			if $z {
				echo "z is true"
			} elif $y {
				echo "y is true"
			} elif $x {
				echo "x is true"
			}`, want: "x is true\n(nil)\n"},
		{desc: "single then, two elif, and else, expecting else", expr: `
			if $z {
				echo "z is true"
			} elif $y {
				echo "y is true"
			} elif $x {
				echo "x is true"
			} else {
				echo "none is true"
			}`, want: "none is true\n(nil)\n"},
		{desc: "compressed then", expr: `set x "Hello" ; if $x { echo "true" }`, want: "true\n(nil)\n"},
		{desc: "compressed else", expr: `if $x { echo "true" } else { echo "false" }`, want: "false\n(nil)\n"},
		{desc: "compressed if", expr: `if $x { echo "x" } elif $y { echo "y" } else { echo "false" }`, want: "false\n(nil)\n"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())
			err := inst.EvalAndDisplay(ctx, tt.expr)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, outW.String())
		})
	}
}

func TestBuiltins_ForEach(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want string
	}{
		{desc: "iterate over list", expr: `
			foreach ["1" "2" "3"] { |v|
				echo $v
			}`, want: "1\n2\n3\n(nil)\n"},
		// TODO: hash is not sorted, so need to find a way to sort it
		{desc: "iterate over map", expr: `
			foreach [a:"1"] { |k v| echo $k "=" $v }`, want: "a=1\n(nil)\n"},
		{desc: "iterate via pipe", expr: `["2" "4" "6"] | foreach { |x| echo $x }`, want: "2\n4\n6\n(nil)\n"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())
			err := inst.EvalAndDisplay(ctx, tt.expr)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, outW.String())
		})
	}
}

func TestBuiltins_Procs(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want string
	}{
		{desc: "simple procs", expr: `
			proc greet {
				echo "Hello, world"
			}

			greet
			greet`, want: "Hello, world\nHello, world\n(nil)\n"},
		{desc: "multiple procs", expr: `
			proc greet { |what|
				echo "Hello, " $what
			}
			proc greetWorld { greet "world" }
			proc greetMoon { greet "moon" }
			proc greetTheThing { |what| greet (cat "the " $what) }

			greetWorld
			greetMoon
			greetTheThing "sun"
			`, want: "Hello, world\nHello, moon\nHello, the sun\n(nil)\n"},
		{desc: "recursive procs", expr: `
			proc four4 { |xs|
				if (eq $xs "xxxx") {
					$xs
				} else {
					four4 (cat $xs "x")
				}
			}

			four4
			`, want: "xxxx\n"},
		{desc: "closures", expr: `
			proc makeGreeter { |greeting|
				proc { |what|
					echo $greeting ", " $what
				}
			}

			set helloGreater (makeGreeter "Hello")
			$helloGreater "world"

			set goodbye (makeGreeter "Goodbye cruel")
			$goodbye "world"

			call (makeGreeter "Quick") "call me"

			`, want: "Hello, world\nGoodbye cruel, world\nQuick, call me\n(nil)\n"},
		//{desc: "modifying closed over variables", expr: `
		//	proc makeSetter {
		//		set bla "X"
		//		proc appendToBla { |x|
		//			set bla (cat $bla $x)
		//		}
		//	}
		//
		//	set er (makeSetter)
		//	call $er "xxx"
		//	call $er "yyy"
		//	`, want: "Xxxx\nXxxxyyy(nil)\n"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())
			err := inst.EvalAndDisplay(ctx, tt.expr)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, outW.String())
		})
	}
}

func TestBuiltins_Map(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want string
	}{
		{desc: "map list", expr: `
			proc makeUpper { |x| $x | toUpper }

			map ["a" "b" "c"] (proc { |x| makeUpper $x }) 
			`, want: "[A B C]\n"},
		{desc: "map list 2", expr: `
			set makeUpper (proc { |x| $x | toUpper })

			map ["a" "b" "c"] $makeUpper 
			`, want: "[A B C]\n"},
		{desc: "map list with pipe", expr: `
			set makeUpper (proc { |x| $x | toUpper })

			["a" "b" "c"] | map $makeUpper 
			`, want: "[A B C]\n"},
		{desc: "map list with block", expr: `
			map ["a" "b" "c"] { |x| toUpper $x } 
			`, want: "[A B C]\n"},
		//{desc: "map list with stream", expr: `
		//	set makeUpper (proc { |x| $x | toUpper })
		//
		//	set l (["a" "b" "c"] | map $makeUpper)
		//	echo $l
		//	`, want: "[A B C]\n"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())
			err := inst.EvalAndDisplay(ctx, tt.expr)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, outW.String())
		})
	}
}

func TestBuiltins_Index(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want string
	}{
		{desc: "index from list 1", expr: `index ["alpha" "beta" "gamma"] 0`, want: "alpha\n"},
		{desc: "index from list 2", expr: `index ["alpha" "beta" "gamma"] 1`, want: "beta\n"},
		{desc: "index from list 3", expr: `index ["alpha" "beta" "gamma"] 2`, want: "gamma\n"},
		{desc: "index from list 4", expr: `index ["alpha" "beta" "gamma"] 3`, want: "(nil)\n"},

		{desc: "index from hash 1", expr: `index ["first":"alpha" "second":"beta" "third":"gamma"] "first"`, want: "alpha\n"},
		{desc: "index from hash 2", expr: `index ["first":"alpha" "second":"beta" "third":"gamma"] "second"`, want: "beta\n"},
		{desc: "index from hash 3", expr: `index ["first":"alpha" "second":"beta" "third":"gamma"] "third"`, want: "gamma\n"},
		{desc: "index from hash 4", expr: `index ["first":"alpha" "second":"beta" "third":"gamma"] "missing"`, want: "(nil)\n"},

		{desc: "multi-list 1", expr: `index [[1 2] [3 4]] 0 1`, want: "2\n"},
		{desc: "multi-list 2", expr: `index [[1 2] [3 4]] 1 0`, want: "3\n"},
		{desc: "list of hash 1", expr: `index [["id":"abc"] ["id":"123"]] 0 id`, want: "abc\n"},
		{desc: "list of hash 2", expr: `index [["id":"abc"] ["id":"123"]] 1 id`, want: "123\n"},

		{desc: "go list 1", expr: `goInt | index 1`, want: "5\n"},
		{desc: "go list 2", expr: `goInt | index 2`, want: "4\n"},
		{desc: "go struct 1", expr: `goStruct | index Alpha`, want: "foo\n"},
		{desc: "go struct 2", expr: `goStruct | index Beta`, want: "bar\n"},
		{desc: "go struct 3", expr: `goStruct | index Gamma 1`, want: "33\n"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())
			inst.SetBuiltin("goInt", func(ctx context.Context, args CallArgs) (any, error) {
				return []int{6, 5, 4}, nil
			})
			inst.SetBuiltin("goStruct", func(ctx context.Context, args CallArgs) (any, error) {
				return struct {
					Alpha string
					Beta  string
					Gamma []int
				}{
					Alpha: "foo",
					Beta:  "bar",
					Gamma: []int{22, 33},
				}, nil
			})
			err := inst.EvalAndDisplay(ctx, tt.expr)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, outW.String())
		})
	}
}

func TestBuiltins_Len(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want string
	}{
		{desc: "len of list 1", expr: `len ["alpha" "beta" "gamma"]`, want: "3\n"},
		{desc: "len of list 2", expr: `len ["alpha"]`, want: "1\n"},
		{desc: "len of list 3", expr: `len []`, want: "0\n"},

		{desc: "len of hash 1", expr: `len ["first":"alpha" "second":"beta" "third":"gamma"]`, want: "3\n"},
		{desc: "len of hash 2", expr: `len ["first":"alpha" "second":"beta"]`, want: "2\n"},
		{desc: "len of hash 3", expr: `len ["first":"alpha"]`, want: "1\n"},
		{desc: "len of hash 4", expr: `len [:]`, want: "0\n"},

		{desc: "len of string 1", expr: `len "Hello, world"`, want: "12\n"},
		{desc: "len of string 2", expr: `len "chair"`, want: "5\n"},
		{desc: "len of string 3", expr: `len ""`, want: "0\n"},

		{desc: "len of int", expr: `len 1232`, want: "0\n"},
		{desc: "len of nil", expr: `len ()`, want: "0\n"},

		{desc: "go list 1", expr: `goInt | len`, want: "3\n"},
		{desc: "go struct 1", expr: `goStruct | len`, want: "3\n"},
		{desc: "go struct 2", expr: `index (goStruct) Gamma | len`, want: "2\n"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())
			inst.SetBuiltin("goInt", func(ctx context.Context, args CallArgs) (any, error) {
				return []int{6, 5, 4}, nil
			})
			inst.SetBuiltin("goStruct", func(ctx context.Context, args CallArgs) (any, error) {
				return struct {
					Alpha   string
					Beta    string
					Gamma   []int
					hidden  string
					missing string
				}{
					Alpha:   "foo",
					Beta:    "bar",
					Gamma:   []int{22, 33},
					hidden:  "hidden",
					missing: "missing",
				}, nil
			})
			err := inst.EvalAndDisplay(ctx, tt.expr)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, outW.String())
		})
	}
}

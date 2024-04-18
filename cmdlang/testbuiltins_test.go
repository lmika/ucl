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

		i.rootEC.addCmd("pipe", invokableFunc(func(ctx context.Context, args invocationArgs) (object, error) {
			return &listIterStream{
				list: args.args,
			}, nil
		}))

		i.rootEC.addCmd("joinpipe", invokableStreamFunc(func(ctx context.Context, inStream stream, args invocationArgs) (object, error) {
			sb := strings.Builder{}
			if err := forEach(inStream, func(o object, i int) error {
				if i > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(o.String())
				return nil
			}); err != nil {
				return nil, err
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
				}
				four4 (cat $xs "x")
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
		{desc: "modifying closed over variables", expr: `
			proc makeSetter {
				set bla "X"
				proc appendToBla { |x|
					set bla (cat $bla $x)
				}
			}
			
			set er (makeSetter)
			call $er "xxx"
			call $er "yyy"
			`, want: "Xxxx\nXxxxyyy(nil)\n"},
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
			`, want: "A\nB\nC\n"},
		{desc: "map list 2", expr: `
			set makeUpper (proc { |x| $x | toUpper })

			map ["a" "b" "c"] $makeUpper 
			`, want: "A\nB\nC\n"},
		{desc: "map list with stream", expr: `
			set makeUpper (proc { |x| $x | toUpper })

			["a" "b" "c"] | map $makeUpper 
			`, want: "A\nB\nC\n"},
		{desc: "map list with stream", expr: `
			set makeUpper (proc { |x| $x | toUpper })
		
			set l (["a" "b" "c"] | map $makeUpper)
			echo $l
			`, want: "[A B C]\n"},
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

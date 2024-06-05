package ucl

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

		i.rootEC.setOrDefineVar("a", strObject("alpha"))
		i.rootEC.setOrDefineVar("bee", strObject("buzz"))
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
		{desc: "multi-line 4", expr: `
# This is a comment
#

;;;
# This is another comment
			echo "Hello"
;

			echo "world"	# command after this
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
			err := EvalAndDisplay(ctx, inst, tt.expr)

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
			err := EvalAndDisplay(ctx, inst, tt.expr)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, outW.String())
		})
	}
}

func TestBuiltins_Break(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want string
	}{
		{desc: "break unconditionally returning nothing", expr: `
			foreach ["1" "2" "3"] { |v|
				break
				echo $v
			}`, want: "(nil)\n"},
		{desc: "break conditionally returning nothing", expr: `
			foreach ["1" "2" "3"] { |v|
				echo $v
				if (eq $v "2") { break }
			}`, want: "1\n2\n(nil)\n"},
		{desc: "break inner loop only returning nothing", expr: `
			foreach ["a" "b"] { |u|
				foreach ["1" "2" "3"] { |v|
					echo $u $v
					if (eq $v "2") { break }
				}
			}`, want: "a1\na2\nb1\nb2\n(nil)\n"},
		{desc: "break returning value", expr: `
			echo (foreach ["1" "2" "3"] { |v|
				echo $v
				if (eq $v "2") { break "hello" }
			})`, want: "1\n2\nhello\n(nil)\n"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())
			err := EvalAndDisplay(ctx, inst, tt.expr)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, outW.String())
		})
	}
}

func TestBuiltins_Continue(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want string
	}{
		{desc: "continue unconditionally", expr: `
			foreach ["1" "2" "3"] { |v|
				echo $v "s"
				continue
				echo $v "e"
			}`, want: "1s\n2s\n3s\n(nil)\n"},
		{desc: "conditionally conditionally", expr: `
			foreach ["1" "2" "3"] { |v|
				echo $v "s"
				if (eq $v "2") { continue }
				echo $v "e"
			}`, want: "1s\n1e\n2s\n3s\n3e\n(nil)\n"},
		{desc: "continue inner loop only", expr: `
			foreach ["a" "b"] { |u|
				foreach ["1" "2" "3"] { |v|	
					if (eq $v "2") { continue }
					echo $u $v
				}
			}`, want: "a1\na3\nb1\nb3\n(nil)\n"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())
			err := EvalAndDisplay(ctx, inst, tt.expr)

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
		{desc: "modifying closed over variables", expr: `
			proc makeSetter {
				set bla "X"
				proc appendToBla { |x|
					set bla (cat $bla $x)
				}
			}
		
			set er (makeSetter)
			echo (call $er "xxx")
			echo (call $er "yyy")
			`, want: "Xxxx\nXxxxyyy\n(nil)\n"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())
			err := EvalAndDisplay(ctx, inst, tt.expr)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, outW.String())
		})
	}
}

func TestBuiltins_Return(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want string
	}{
		// syntax tests
		{desc: "empty proc 1", expr: `
			proc greet {}
			greet
		`, want: "(nil)\n"},
		{desc: "empty proc 2", expr: `
			proc greet {
			}

			greet
		`, want: "(nil)\n"},
		{desc: "empty proc 3", expr: `
			proc greet {


			}

			greet
		`, want: "(nil)\n"},
		{desc: "empty proc 4", expr: `
			proc greet {
				# bla
				
				# di
				# bla!
			}

			greet
		`, want: "(nil)\n"},

		{desc: "nil return", expr: `
			proc greet {
				echo "Hello"
				return
				echo "World"
			}

			greet
			`, want: "Hello\n(nil)\n"},

		{desc: "simple arg 1", expr: `
			proc greet { |x|
				return (cat "Hello, " $x)
			}

			greet "person"
			`, want: "Hello, person\n"},
		{desc: "simple arg 2", expr: `
			proc greet { 
				# This will greet someone
				# here are the args:
				|x|

				# And here is the code
				return (cat "Hello, " $x)
			}

			greet "person"
			`, want: "Hello, person\n"},

		{desc: "simple return", expr: `
			proc greet {
				return "Hello, world"
				echo "But not me"
			}

			greet
			`, want: "Hello, world\n"},

		{desc: "only return current frame", expr: `
			proc greetWhat {
				echo "Greet the"
				return "moon"
				echo "world"
			}
			proc greet {
				set what (greetWhat)
				echo "Hello, " $what
			}

			greet
			`, want: "Greet the\nHello, moon\n(nil)\n"},
		{desc: "return in loop", expr: `
			proc countdown { |nums|
				foreach $nums { |n|
					echo $n
					if (eq $n 3) {
						return "abort"
					}
				}
			}
			countdown [5 4 3 2 1]
			`, want: "5\n4\n3\nabort\n"},
		{desc: "recursive procs", expr: `
			proc four4 { |xs|
				if (eq $xs "xxxx") {
					return $xs
				}
				four4 (cat $xs "x")	
			}

			four4
			`, want: "xxxx\n"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())
			err := EvalAndDisplay(ctx, inst, tt.expr)

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
		{desc: "map list with pipe", expr: `
			set makeUpper (proc { |x| $x | toUpper })

			["a" "b" "c"] | map $makeUpper 
			`, want: "A\nB\nC\n"},
		{desc: "map list with block", expr: `
			map ["a" "b" "c"] { |x| toUpper $x } 
			`, want: "A\nB\nC\n"},
		{desc: "map list with stream", expr: `
			set makeUpper (proc { |x| toUpper $x })
		
			set l (["a" "b" "c"] | map $makeUpper)
			echo $l
			`, want: "[A B C]\n(nil)\n"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())
			err := EvalAndDisplay(ctx, inst, tt.expr)

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

		{desc: "go int 1", expr: `goInt | index 1`, want: "5\n"},
		{desc: "go int 2", expr: `goInt | index 2`, want: "4\n"},
		{desc: "go int 3", expr: `goInt | index 555`, want: "(nil)\n"},
		{desc: "go int 4", expr: `goInt | index -12`, want: "(nil)\n"},
		{desc: "go int 5", expr: `goInt | index NotAnIndex`, want: "(nil)\n"},

		{desc: "go list 1", expr: `goList | index 0 This`, want: "thing 1\n"},
		{desc: "go list 2", expr: `goList | index 1 This`, want: "thing 2\n"},
		{desc: "go list 3", expr: `goList | index 2`, want: "(nil)\n"},
		{desc: "go list 4", expr: `goList | index 2 This`, want: "(nil)\n"},
		{desc: "go list 5", expr: `goList | index 30`, want: "(nil)\n"},

		{desc: "go struct 1", expr: `goStruct | index Alpha`, want: "foo\n"},
		{desc: "go struct 2", expr: `goStruct | index Beta`, want: "bar\n"},
		{desc: "go struct 3", expr: `goStruct | index Gamma 1`, want: "33\n"},
		{desc: "go struct 4", expr: `goStruct | index Nested This`, want: "fla\n"},
		{desc: "go struct 5", expr: `goStruct | index Nested That`, want: "132\n"},
		{desc: "go struct 6", expr: `goStruct | index NestedPtr This`, want: "flaPtr\n"},
		{desc: "go struct 7", expr: `goStruct | index NestedPtr That`, want: "6678\n"},
		{desc: "go struct 8", expr: `goStruct | index Missing`, want: "(nil)\n"},
		{desc: "go struct 9", expr: `goStruct | index Nested Missing 123 Stuff`, want: "(nil)\n"},
		{desc: "go struct 10", expr: `goStruct | index NestedPtrNil`, want: "(nil)\n"},
		{desc: "go struct 11", expr: `goStruct | index NestedPtrNil This`, want: "(nil)\n"},
		{desc: "go struct 12", expr: `goStruct | index NestedPtrNil Missing`, want: "(nil)\n"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())
			inst.SetBuiltin("goInt", func(ctx context.Context, args CallArgs) (any, error) {
				return []int{6, 5, 4}, nil
			})
			inst.SetBuiltin("goList", func(ctx context.Context, args CallArgs) (any, error) {
				type nest struct {
					This string
				}
				return []*nest{
					{This: "thing 1"},
					{This: "thing 2"},
					nil,
				}, nil
			})
			inst.SetBuiltin("goStruct", func(ctx context.Context, args CallArgs) (any, error) {
				type nested struct {
					This string
					That int
				}
				return struct {
					Alpha        string
					Beta         string
					Gamma        []int
					Nested       nested
					NestedPtr    *nested
					NestedPtrNil *nested
				}{
					Alpha: "foo",
					Beta:  "bar",
					Gamma: []int{22, 33},
					Nested: nested{
						This: "fla",
						That: 132,
					},
					NestedPtr: &nested{
						This: "flaPtr",
						That: 6678,
					},
				}, nil
			})
			err := EvalAndDisplay(ctx, inst, tt.expr)

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
			err := EvalAndDisplay(ctx, inst, tt.expr)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, outW.String())
		})
	}
}

func TestBuiltins_Keys(t *testing.T) {
	type testNested struct {
		Nested string
		Type   string
	}

	tests := []struct {
		desc      string
		expr      string
		wantItems []string
	}{
		{desc: "keys of map", expr: `keys [alpha: "hello" bravo: "world"]`, wantItems: []string{"alpha", "bravo"}},
		{desc: "keys of go struct 1", expr: `goStruct | keys`, wantItems: []string{"Alpha", "Beta", "Gamma"}},
		{desc: "keys of go struct 2", expr: `index (goStruct) Gamma | keys`, wantItems: []string{"Nested", "Type"}},
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
					Gamma   testNested
					hidden  string
					missing string
				}{
					Alpha: "foo",
					Beta:  "bar",
					Gamma: testNested{
						Nested: "ads",
						Type:   "asd",
					},
					hidden:  "hidden",
					missing: "missing",
				}, nil
			})

			res, err := inst.Eval(ctx, tt.expr)
			assert.NoError(t, err)
			assert.Len(t, res, len(tt.wantItems))
			for _, i := range tt.wantItems {
				assert.Contains(t, res, i)
			}
		})
	}
}

func TestBuiltins_Filter(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want any
	}{
		{desc: "filter list 1", expr: `filter [1 2 3] { |x| eq $x 2 }`, want: []any{2}},
		{desc: "filter list 2", expr: `filter ["flim" "flam" "fla"] { |x| eq $x "flam" }`, want: []any{"flam"}},
		{desc: "filter list 3", expr: `filter ["flim" "flam" "fla"] { |x| eq $x "bogie" }`, want: []any{}},

		{desc: "filter map 1", expr: `filter [alpha:"hello" bravo:"world"] { |k v| eq $k "alpha" }`, want: map[string]any{
			"alpha": "hello",
		}},
		{desc: "filter map 2", expr: `filter [alpha:"hello" bravo:"world"] { |k v| eq $v "world" }`, want: map[string]any{
			"bravo": "world",
		}},
		{desc: "filter map 3", expr: `filter [alpha:"hello" bravo:"world"] { |k v| eq $v "alpha" }`, want: map[string]any{}},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())

			res, err := inst.Eval(ctx, tt.expr)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

func TestBuiltins_Reduce(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want any
	}{
		{desc: "reduce list 1", expr: `reduce [1 1 1] { |x a| add $x $a }`, want: 3},
		{desc: "reduce list 2", expr: `reduce [1 1 1] 20 { |x a| add $x $a }`, want: 23},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := New(WithOut(outW), WithTestBuiltin())

			res, err := inst.Eval(ctx, tt.expr)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

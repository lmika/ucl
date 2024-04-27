package ucl_test

import (
	"bytes"
	"context"
	"testing"
	"ucl.lmika.dev/ucl"

	"github.com/stretchr/testify/assert"
)

func TestInst_Eval(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want any
	}{
		{desc: "simple string", expr: `firstarg "hello"`, want: "hello"},
		{desc: "simple int 1", expr: `firstarg 123`, want: 123},
		{desc: "simple int 2", expr: `firstarg -234`, want: -234},
		{desc: "simple ident", expr: `firstarg a-test`, want: "a-test"},

		// Sub-expressions
		{desc: "sub expression 1", expr: `firstarg (sjoin "hello")`, want: "hello"},
		{desc: "sub expression 2", expr: `firstarg (sjoin "hello " "world")`, want: "hello world"},
		{desc: "sub expression 3", expr: `firstarg (sjoin "hello" (sjoin " ") (sjoin "world"))`, want: "hello world"},

		// Variables
		{desc: "var 1", expr: `firstarg $a`, want: "alpha"},
		{desc: "var 2", expr: `firstarg $bee`, want: "buzz"},
		{desc: "var 3", expr: `firstarg (sjoin $bee " " $bee " " $bee)`, want: "buzz buzz buzz"},

		// Pipeline
		{desc: "pipe 1", expr: `list "aye" "bee" "see" | joinpipe`, want: "aye,bee,see"},
		{desc: "pipe 2", expr: `list "aye" "bee" "see" | map { |x| toUpper $x } | joinpipe`, want: "AYE,BEE,SEE"},
		{desc: "pipe 3", expr: `firstarg ["normal"] | map { |x| toUpper $x } | joinpipe`, want: "NORMAL"},
		{desc: "pipe literal 1", expr: `"hello" | firstarg`, want: "hello"},
		{desc: "pipe literal 2", expr: `["hello" "world"] | joinpipe`, want: "hello,world"},

		{desc: "ignored pipe", expr: `(list "aye" | firstarg "ignore me") | joinpipe`, want: "aye"},

		// Multi-statements
		{desc: "multi 1", expr: `firstarg "hello" ; firstarg "world"`, want: "world"},
		{desc: "multi 2", expr: `list "hello" | toUpper ; firstarg "world"`, want: "world"},
		{desc: "multi 3", expr: `set new "this is new" ; firstarg $new`, want: "this is new"},

		// Lists
		{desc: "list 1", expr: `firstarg ["1" "2" "3"]`, want: []any{"1", "2", "3"}},
		{desc: "list 2", expr: `set one "one" ; firstarg [$one (list "two" | map { |x| toUpper $x } | head) "three"]`, want: []any{"one", "TWO", "three"}},
		{desc: "list 3", expr: `firstarg []`, want: []any{}},

		// Maps
		{desc: "map 1", expr: `firstarg [one:"1" two:"2" three:"3"]`, want: map[string]any{"one": "1", "two": "2", "three": "3"}},
		{desc: "map 2", expr: `firstarg ["one":"1" "two":"2" "three":"3"]`, want: map[string]any{"one": "1", "two": "2", "three": "3"}},
		{desc: "map 3", expr: `
			set one "one" ; set n1 "1"
			firstarg [
				$one:$n1
				(list "two" | map { |x| toUpper $x } | head):(list "2" | map { |x| toUpper $x } | head)
				three:"3"
			]`, want: map[string]any{"one": "1", "TWO": "2", "three": "3"}},
		{desc: "map 4", expr: `firstarg [:]`, want: map[string]any{}},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := ucl.New(ucl.WithOut(outW), ucl.WithTestBuiltin())
			res, err := inst.Eval(ctx, tt.expr)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

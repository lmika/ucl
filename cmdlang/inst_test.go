package cmdlang_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/lmika/cmdlang-proto/cmdlang"
	"github.com/stretchr/testify/assert"
)

func TestInst_Eval(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want any
	}{
		{desc: "simple string", expr: `firstarg "hello"`, want: "hello"},
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
		{desc: "pipe 1", expr: `pipe "aye" "bee" "see" | joinpipe`, want: "aye,bee,see"},
		{desc: "pipe 2", expr: `pipe "aye" "bee" "see" | toUpper | joinpipe`, want: "AYE,BEE,SEE"},
		{desc: "pipe 3", expr: `firstarg "normal" | toUpper | joinpipe`, want: "NORMAL"},

		{desc: "ignored pipe", expr: `pipe "aye" "bee" "see" | firstarg "ignore me"`, want: "ignore me"}, // TODO: check for leaks

		// Multi-statements
		{desc: "multi 1", expr: `firstarg "hello" ; firstarg "world"`, want: "world"},
		{desc: "multi 2", expr: `pipe "hello" | toUpper ; firstarg "world"`, want: "world"}, // TODO: assert for leaks
		{desc: "multi 3", expr: `set new "this is new" ; firstarg $new`, want: "this is new"},

		// Lists
		{desc: "list 1", expr: `firstarg ["1" "2" "3"]`, want: []any{"1", "2", "3"}},
		{desc: "list 2", expr: `set one "one" ; firstarg [$one (pipe "two" | toUpper | head) "three"]`, want: []any{"one", "TWO", "three"}},
		{desc: "list 3", expr: `firstarg []`, want: []any{}},

		// Maps
		{desc: "map 1", expr: `firstarg [one:"1" two:"2" three:"3"]`, want: map[string]any{"one": "1", "two": "2", "three": "3"}},
		{desc: "map 2", expr: `firstarg ["one":"1" "two":"2" "three":"3"]`, want: map[string]any{"one": "1", "two": "2", "three": "3"}},
		{desc: "map 3", expr: `
			set one "one" ; set n1 "1"
			firstarg [
				$one:$n1
				(firstarg "two" | toUpper | head):(firstarg "2" | toUpper | head)
				three:"3"
			]`, want: map[string]any{"one": "1", "TWO": "2", "three": "3"}},
		{desc: "map 4", expr: `firstarg [:]`, want: map[string]any{}},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ctx := context.Background()
			outW := bytes.NewBuffer(nil)

			inst := cmdlang.New(cmdlang.WithOut(outW), cmdlang.WithTestBuiltin())
			res, err := inst.Eval(ctx, tt.expr)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

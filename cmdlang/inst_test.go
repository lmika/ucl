package cmdlang_test

import (
	"bytes"
	"context"
	"github.com/lmika/cmdlang-proto/cmdlang"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInst_Eval(t *testing.T) {
	tests := []struct {
		desc string
		expr string
		want string
	}{
		{desc: "simple string", expr: `firstarg "hello"`, want: "hello"},

		// Sub-expressions
		{desc: "sub expression 1", expr: `firstarg (echo "hello")`, want: "hello"},
		{desc: "sub expression 2", expr: `firstarg (echo "hello " "world")`, want: "hello world"},
		{desc: "sub expression 3", expr: `firstarg (echo "hello" (echo " ") (echo "world"))`, want: "hello world"},

		// Variables
		{desc: "var 1", expr: `firstarg $a`, want: "alpha"},
		{desc: "var 2", expr: `firstarg $bee`, want: "buzz"},
		{desc: "var 3", expr: `firstarg (echo $bee " " $bee " " $bee)`, want: "buzz buzz buzz"},

		// Pipeline
		{desc: "pipe 1", expr: `pipe "aye" "bee" "see" | joinpipe`, want: "aye,bee,see"},
		{desc: "pipe 2", expr: `pipe "aye" "bee" "see" | toUpper | joinpipe`, want: "AYE,BEE,SEE"},
		{desc: "pipe 3", expr: `firstarg "normal" | toUpper | joinpipe`, want: "NORMAL"},

		{desc: "ignored pipe", expr: `pipe "aye" "bee" "see" | firstarg "ignore me"`, want: "ignore me"}, // TODO: check for leaks

		// Multi-statements
		{desc: "multi 1", expr: `firstarg "hello" ; firstarg "world"`, want: "world"},
		{desc: "multi 2", expr: `pipe "hello" | toUpper ; firstarg "world"`, want: "world"}, // TODO: assert for leaks
		{desc: "multi 3", expr: `set new "this is new" ; firstarg $new`, want: "this is new"},
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

func TestInst_Builtins(t *testing.T) {
	t.Run("echo", func(t *testing.T) {
		tests := []struct {
			desc string
			expr string
			want string
		}{
			{desc: "no args", expr: `echo`, want: "\n"},
			{desc: "single arg", expr: `echo "hello"`, want: "hello\n"},
			{desc: "dual args", expr: `echo "hello " "world"`, want: "hello world\n"},
			{desc: "args to singleton stream", expr: `echo "aye" "bee" "see" | toUpper`, want: "AYEBEESEE\n"},
		}

		for _, tt := range tests {
			t.Run(tt.desc, func(t *testing.T) {
				ctx := context.Background()
				outW := bytes.NewBuffer(nil)

				inst := cmdlang.New(cmdlang.WithOut(outW), cmdlang.WithTestBuiltin())
				err := inst.EvalAndDisplay(ctx, tt.expr)

				assert.NoError(t, err)
				assert.Equal(t, tt.want, outW.String())
			})
		}
	})
}

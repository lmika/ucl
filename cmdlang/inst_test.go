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

			inst := cmdlang.New(cmdlang.WithOut(outW), cmdlang.WithTestBuiltin())
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

			inst := cmdlang.New(cmdlang.WithOut(outW), cmdlang.WithTestBuiltin())
			err := inst.EvalAndDisplay(ctx, tt.expr)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, outW.String())
		})
	}
}

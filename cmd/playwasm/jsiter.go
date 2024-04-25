//go:build js && wasm

package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/alecthomas/participle/v2"
	"github.com/lmika/cmdlang-proto/cmdlang"
	"strings"
	"syscall/js"
)

func invokeUCLCallback(name string, args ...any) {
	if onLine := js.Global().Get("ucl").Get(name); !onLine.IsNull() {
		onLine.Invoke(args...)
	}
}

func initJS(ctx context.Context) {
	ucl := make(map[string]any)

	inst := cmdlang.New(cmdlang.WithOut(&uclOut{
		lineBuffer: new(bytes.Buffer),
		writeLine: func(line string) {
			invokeUCLCallback("onOutLine", line)
		},
	}))

	ucl["eval"] = js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 2 {
			return nil
		}

		cmdLine := args[0].String()
		if strings.TrimSpace(cmdLine) == "" {
			invokeUCLCallback("onNewCommand")
			return nil
		}

		wantContinue := args[1].Bool()
		if err := inst.EvalAndDisplay(ctx, cmdLine); err != nil {
			var p participle.Error
			if errors.As(err, &p) && wantContinue {
				invokeUCLCallback("onContinue")
				return nil
			}

			invokeUCLCallback("onError", err.Error())
		}
		invokeUCLCallback("onNewCommand")
		return nil
	})
	js.Global().Set("ucl", ucl)
}

type uclOut struct {
	lineBuffer *bytes.Buffer
	writeLine  func(line string)
}

func (uo *uclOut) Write(p []byte) (n int, err error) {
	for _, b := range p {
		if b == '\n' {
			uo.writeLine(uo.lineBuffer.String())
			uo.lineBuffer.Reset()
		} else {
			uo.lineBuffer.WriteByte(b)
		}
	}
	return len(p), nil
}

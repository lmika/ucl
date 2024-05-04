package main

import (
	"context"
	"github.com/chzyer/readline"
	"log"
	"ucl.lmika.dev/ucl"
	"ucl.lmika.dev/ucl/builtins"
)

func main() {
	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	inst := ucl.New(
		ucl.WithModule(builtins.OS()),
		ucl.WithModule(builtins.FS(nil)),
	)
	ctx := context.Background()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}

		if err := ucl.EvalAndDisplay(ctx, inst, line); err != nil {
			log.Printf("%T: %v", err, err)
		}
	}
}

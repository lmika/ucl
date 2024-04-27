package main

import (
	"context"
	"github.com/chzyer/readline"
	"github.com/lmika/ucl/ucl"
	"log"
)

func main() {
	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	inst := ucl.New()
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

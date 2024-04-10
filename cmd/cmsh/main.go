package main

import (
	"context"
	"github.com/chzyer/readline"
	"github.com/lmika/cmdlang-proto/cmdlang"
	"log"
)

func main() {
	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	inst := cmdlang.New()
	ctx := context.Background()

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}

		if err := inst.EvalAndDisplay(ctx, line); err != nil {
			log.Printf("%T: %v", err, err)
		}
	}
}

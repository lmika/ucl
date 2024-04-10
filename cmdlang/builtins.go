package cmdlang

import (
	"context"
	"log"
	"strings"
)

func echoBuiltin(ctx context.Context, args invocationArgs) error {
	if len(args.args) == 0 {
		log.Print()
		return nil
	}

	var line strings.Builder
	for _, arg := range args.args {
		line.WriteString(arg)
	}
	log.Print(line.String())
	return nil
}

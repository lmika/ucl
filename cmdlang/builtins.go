package cmdlang

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func echoBuiltin(ctx context.Context, args invocationArgs) (object, error) {
	if len(args.args) == 0 {
		if _, err := fmt.Fprintln(args.inst.Out()); err != nil {
			return nil, err
		}
		return nil, nil
	}

	var line strings.Builder
	for _, arg := range args.args {
		if s, ok := arg.(fmt.Stringer); ok {
			line.WriteString(s.String())
		}
	}

	if _, err := fmt.Fprintln(args.inst.Out(), line.String()); err != nil {
		return nil, err
	}
	return nil, nil
}

func setBuiltin(ctx context.Context, args invocationArgs) (object, error) {
	if err := args.expectArgn(2); err != nil {
		return nil, err
	}

	name, err := args.stringArg(0)
	if err != nil {
		return nil, err
	}

	newVal := args.args[1]

	// TODO: if the value is a stream, consume the stream and save it as a list
	args.ec.setVar(name, newVal)
	return newVal, nil
}

func toUpperBuiltin(ctx context.Context, inStream stream, args invocationArgs) (object, error) {
	// Handle args
	return mapFilterStream{
		in: inStream,
		mapFn: func(x object) (object, bool) {
			s, ok := x.(strObject)
			if !ok {
				return nil, false
			}
			return strObject(strings.ToUpper(string(s))), true
		},
	}, nil
}

func catBuiltin(ctx context.Context, args invocationArgs) (object, error) {
	if err := args.expectArgn(1); err != nil {
		return nil, err
	}

	filename, err := args.stringArg(0)
	if err != nil {
		return nil, err
	}

	return &fileLinesStream{filename: filename}, nil
}

type fileLinesStream struct {
	filename string
	f        *os.File
	scnr     *bufio.Scanner
}

func (f *fileLinesStream) String() string {
	return fmt.Sprintf("fileLinesStream{file: %v}", f.filename)
}

func (f *fileLinesStream) Truthy() bool {
	return true // ??
}

func (f *fileLinesStream) next() (object, error) {
	var err error

	// We open the file on the first pull. That way, an unconsumed stream won't result in a FD leak
	if f.f == nil {
		f.f, err = os.Open(f.filename)
		if err != nil {
			return nil, err
		}
		f.scnr = bufio.NewScanner(f.f)
	}

	if f.scnr.Scan() {
		return strObject(f.scnr.Text()), nil
	}
	if f.scnr.Err() == nil {
		return nil, io.EOF
	}
	return nil, f.scnr.Err()
}

func (f *fileLinesStream) close() error {
	if f.f != nil {
		return f.f.Close()
	}
	return nil
}

func ifBuiltin(ctx context.Context, args macroArgs) (object, error) {
	if args.nargs() < 2 {
		return nil, errors.New("need at least 2 arguments")
	}

	if guard, err := args.evalArg(ctx, 0); err == nil && isTruthy(guard) {
		return args.evalBlock(ctx, 1)
	} else if err != nil {
		return nil, err
	}

	args.shift(2)
	for args.identIs(ctx, 0, "elif") {
		args.shift(1)

		if args.nargs() < 2 {
			return nil, errors.New("need at least 2 arguments")
		}

		if guard, err := args.evalArg(ctx, 0); err == nil && isTruthy(guard) {
			return args.evalBlock(ctx, 1)
		} else if err != nil {
			return nil, err
		}

		args.shift(2)
	}

	if args.identIs(ctx, 0, "else") && args.nargs() > 1 {
		return args.evalBlock(ctx, 1)
	} else if args.nargs() == 0 {
		// no elif or else
		return nil, nil
	}

	return nil, errors.New("malformed if-elif-else")
}

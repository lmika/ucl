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
		return asStream(""), nil
	}

	var line strings.Builder
	for _, arg := range args.args {
		if s, ok := arg.(fmt.Stringer); ok {
			line.WriteString(s.String())
		}
	}

	return asStream(line.String()), nil
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
			s, ok := x.(string)
			if !ok {
				return nil, false
			}
			return strings.ToUpper(s), true
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
		return f.scnr.Text(), nil
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

func errorTestBuiltin(ctx context.Context, inStream stream, args invocationArgs) (object, error) {
	return &timeBombStream{inStream, 2}, nil
}

type timeBombStream struct {
	in stream
	x  int
}

func (ms *timeBombStream) next() (object, error) {
	if ms.x > 0 {
		ms.x--
		return ms.in.next()
	}
	return nil, errors.New("BOOM")
}

func (ms *timeBombStream) close() error {
	return ms.in.close()
}

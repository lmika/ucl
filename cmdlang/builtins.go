package cmdlang

import (
	"bufio"
	"context"
	"errors"
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
		line.WriteString(arg)
	}

	return asStream(line.String()), nil
}

func toUpperBuiltin(ctx context.Context, args invocationArgs) (object, error) {
	// Handle args
	return mapFilterStream{
		in: args.inStream,
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

	return &fileLinesStream{filename: args.args[0]}, nil
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

func errorTestBuiltin(ctx context.Context, args invocationArgs) (object, error) {
	return &timeBombStream{args.inStream, 2}, nil
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
	closable, ok := ms.in.(closableStream)
	if ok {
		return closable.close()
	}
	return nil
}

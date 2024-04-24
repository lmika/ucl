package cmdlang

import (
	"context"
	"errors"
	"fmt"
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

func toUpperBuiltin(ctx context.Context, args invocationArgs) (object, error) {
	if err := args.expectArgn(1); err != nil {
		return nil, err
	}
	sarg, err := args.stringArg(0)
	if err != nil {
		return nil, err
	}
	return strObject(strings.ToUpper(sarg)), nil
}

func eqBuiltin(ctx context.Context, args invocationArgs) (object, error) {
	if err := args.expectArgn(2); err != nil {
		return nil, err
	}

	l := args.args[0]
	r := args.args[1]

	switch lv := l.(type) {
	case strObject:
		if rv, ok := r.(strObject); ok {
			return boolObject(lv == rv), nil
		}
	}
	return boolObject(false), nil
}

func concatBuiltin(ctx context.Context, args invocationArgs) (object, error) {
	var sb strings.Builder

	for _, a := range args.args {
		if a == nil {
			continue
		}
		sb.WriteString(a.String())
	}

	return strObject(sb.String()), nil
}

//
//func catBuiltin(ctx context.Context, args invocationArgs) (object, error) {
//	if err := args.expectArgn(1); err != nil {
//		return nil, err
//	}
//
//	filename, err := args.stringArg(0)
//	if err != nil {
//		return nil, err
//	}
//
//	return &fileLinesStream{filename: filename}, nil
//}

func callBuiltin(ctx context.Context, args invocationArgs) (object, error) {
	if err := args.expectArgn(1); err != nil {
		return nil, err
	}

	inv, ok := args.args[0].(invokable)
	if !ok {
		return nil, errors.New("expected invokable")
	}

	return inv.invoke(ctx, args.shift(1))
}

func indexBuiltin(ctx context.Context, args invocationArgs) (object, error) {
	if err := args.expectArgn(1); err != nil {
		return nil, err
	}

	val := args.args[0]
	for _, idx := range args.args[1:] {
		switch v := val.(type) {
		case listable:
			intIdx, ok := idx.(intObject)
			if !ok {
				return nil, errors.New("expected int for listable")
			}
			if int(intIdx) >= 0 && int(intIdx) < v.Len() {
				val = v.Index(int(intIdx))
			} else {
				val = nil
			}
		case hashable:
			strIdx, ok := idx.(strObject)
			if !ok {
				return nil, errors.New("expected string for hashable")
			}
			val = v.Value(string(strIdx))
		default:
			return val, nil
		}
	}

	return val, nil
}

func mapBuiltin(ctx context.Context, args invocationArgs) (object, error) {
	if err := args.expectArgn(2); err != nil {
		return nil, err
	}

	inv, err := args.invokableArg(1)
	if err != nil {
		return nil, err
	}

	switch t := args.args[0].(type) {
	case listable:
		l := t.Len()
		newList := listObject{}
		for i := 0; i < l; i++ {
			v := t.Index(i)
			m, err := inv.invoke(ctx, args.fork([]object{v}))
			if err != nil {
				return nil, err
			}
			newList = append(newList, m)
		}
		return newList, nil
	}
	return nil, errors.New("expected listable")
}

func firstBuiltin(ctx context.Context, args invocationArgs) (object, error) {
	if err := args.expectArgn(1); err != nil {
		return nil, err
	}

	switch t := args.args[0].(type) {
	case listable:
		if t.Len() == 0 {
			return nil, nil
		}
		return t.Index(0), nil
	}
	return nil, errors.New("expected listable")
}

/*
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
*/

func ifBuiltin(ctx context.Context, args macroArgs) (object, error) {
	if args.nargs() < 2 {
		return nil, errors.New("need at least 2 arguments")
	}

	if guard, err := args.evalArg(ctx, 0); err == nil && isTruthy(guard) {
		return args.evalBlock(ctx, 1, nil, false)
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
			return args.evalBlock(ctx, 1, nil, false)
		} else if err != nil {
			return nil, err
		}

		args.shift(2)
	}

	if args.identIs(ctx, 0, "else") && args.nargs() > 1 {
		return args.evalBlock(ctx, 1, nil, false)
	} else if args.nargs() == 0 {
		// no elif or else
		return nil, nil
	}

	return nil, errors.New("malformed if-elif-else")
}

func foreachBuiltin(ctx context.Context, args macroArgs) (object, error) {
	var (
		items    object
		blockIdx int
		err      error
	)
	if !args.hasPipe {
		if args.nargs() < 2 {
			return nil, errors.New("need at least 2 arguments")
		}

		items, err = args.evalArg(ctx, 0)
		if err != nil {
			return nil, err
		}
		blockIdx = 1
	} else {
		if args.nargs() < 1 {
			return nil, errors.New("need at least 1 argument")
		}
		items = args.pipeArg
		blockIdx = 0
	}

	var last object

	switch t := items.(type) {
	case listable:
		l := t.Len()
		for i := 0; i < l; i++ {
			v := t.Index(i)
			last, err = args.evalBlock(ctx, blockIdx, []object{v}, true) // TO INCLUDE: the index
			if err != nil {
				return nil, err
			}
		}
	case hashObject:
		for k, v := range t {
			last, err = args.evalBlock(ctx, blockIdx, []object{strObject(k), v}, true)
			if err != nil {
				return nil, err
			}
		}
	}

	return last, nil
}

func procBuiltin(ctx context.Context, args macroArgs) (object, error) {
	if args.nargs() < 1 {
		return nil, errors.New("need at least one arguments")
	}

	var procName string
	if args.nargs() == 2 {
		name, ok := args.shiftIdent(ctx)
		if !ok {
			return nil, errors.New("malformed procedure: expected identifier as first argument")
		}
		procName = name
	}

	block, err := args.evalArg(ctx, 0)
	if err != nil {
		return nil, err
	}
	blockObj, ok := block.(blockObject)
	if !ok {
		return nil, fmt.Errorf("malformed procedure: expected block object, was %v", block.String())
	}

	obj := procObject{args.eval, args.ec, blockObj.block}
	if procName != "" {
		args.ec.addCmd(procName, obj)
	}
	return obj, nil
}

type procObject struct {
	eval  evaluator
	ec    *evalCtx
	block *astBlock
}

func (b procObject) String() string {
	return "(proc)"
}

func (b procObject) Truthy() bool {
	return true
}

func (b procObject) invoke(ctx context.Context, args invocationArgs) (object, error) {
	newEc := b.ec.fork()

	for i, name := range b.block.Names {
		if i < len(args.args) {
			newEc.setVar(name, args.args[i])
		} else {
			newEc.setVar(name, nil)
		}
	}

	return b.eval.evalBlock(ctx, newEc, b.block)
}

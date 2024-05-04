package builtins

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"ucl.lmika.dev/ucl"
)

type fsHandlers struct {
	fs fs.FS
}

func FS(fs fs.FS) ucl.Module {
	fsh := fsHandlers{fs: fs}

	return ucl.Module{
		Name: "fs",
		Builtins: map[string]ucl.BuiltinHandler{
			"lines": fsh.lines,
		},
	}
}

func (fh fsHandlers) openFile(name string) (fs.File, error) {
	if fh.fs == nil {
		return os.Open(name)
	}
	return fh.fs.Open(name)
}

func (fh fsHandlers) lines(ctx context.Context, args ucl.CallArgs) (any, error) {
	var fname string
	if err := args.Bind(&fname); err != nil {
		return nil, err
	}

	f, err := fh.openFile(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lines := make([]string, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

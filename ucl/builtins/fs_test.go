package builtins_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"testing/fstest"
	"ucl.lmika.dev/ucl"
	"ucl.lmika.dev/ucl/builtins"
)

var testFS = fstest.MapFS{
	"test.txt": &fstest.MapFile{
		Data: []byte("these\nare\nlines"),
	},
}

func TestFS_Cat(t *testing.T) {
	tests := []struct {
		descr string
		eval  string
		want  any
	}{
		{descr: "read file", eval: `fs:lines "test.txt"`, want: []string{"these", "are", "lines"}},
	}

	for _, tt := range tests {
		t.Run(tt.descr, func(t *testing.T) {
			inst := ucl.New(
				ucl.WithModule(builtins.FS(testFS)),
			)
			res, err := inst.Eval(context.Background(), tt.eval)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

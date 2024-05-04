package builtins_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"ucl.lmika.dev/ucl"
	"ucl.lmika.dev/ucl/builtins"
)

func TestOS_Env(t *testing.T) {
	tests := []struct {
		descr string
		eval  string
		want  any
	}{
		{descr: "env value", eval: `os:env "MY_ENV"`, want: "my env value"},
		{descr: "missing env value", eval: `os:env "MISSING_THING"`, want: ""},
		{descr: "default env value (str)", eval: `os:env "MISSING_THING" "my default"`, want: "my default"},
		{descr: "default env value (int)", eval: `os:env "MISSING_THING" 1352`, want: 1352},
		{descr: "default env value (nil)", eval: `os:env "MISSING_THING" ()`, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.descr, func(t *testing.T) {
			t.Setenv("MY_ENV", "my env value")

			inst := ucl.New(
				ucl.WithModule(builtins.OS()),
			)
			res, err := inst.Eval(context.Background(), tt.eval)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

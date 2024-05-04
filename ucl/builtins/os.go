package builtins

import (
	"context"
	"os"
	"ucl.lmika.dev/ucl"
)

type osHandlers struct {
}

func OS() ucl.Module {
	osh := osHandlers{}

	return ucl.Module{
		Name: "os",
		Builtins: map[string]ucl.BuiltinHandler{
			"env": osh.env,
		},
	}
}

func (oh osHandlers) env(ctx context.Context, args ucl.CallArgs) (any, error) {
	var envName string
	if err := args.Bind(&envName); err != nil {
		return nil, err
	}

	val, ok := os.LookupEnv(envName)
	if ok {
		return val, nil
	}

	var defValue any
	if err := args.Bind(&defValue); err == nil {
		return defValue, nil
	}

	return "", nil
}

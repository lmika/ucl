package ucl

import (
	"context"
	"fmt"
)

func EvalAndDisplay(ctx context.Context, inst *Inst, expr string) error {
	res, err := inst.eval(ctx, expr)
	if err != nil {
		return err
	}

	return displayResult(ctx, inst, res)
}

func displayResult(ctx context.Context, inst *Inst, res object) (err error) {
	switch v := res.(type) {
	case nil:
		if _, err = fmt.Fprintln(inst.out, "(nil)"); err != nil {
			return err
		}
	case listable:
		for i := 0; i < v.Len(); i++ {
			if err = displayResult(ctx, inst, v.Index(i)); err != nil {
				return err
			}
		}
	default:
		if _, err = fmt.Fprintln(inst.out, v.String()); err != nil {
			return err
		}
	}
	return nil
}

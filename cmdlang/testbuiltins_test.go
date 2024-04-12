package cmdlang

import (
	"context"
	"fmt"
	"strings"
)

// Builtins used for test
func WithTestBuiltin() InstOption {
	return func(i *Inst) {
		i.rootEC.addCmd("firstarg", invokableFunc(func(ctx context.Context, args invocationArgs) (object, error) {
			return args.args[0], nil
		}))

		i.rootEC.addCmd("sjoin", invokableFunc(func(ctx context.Context, args invocationArgs) (object, error) {
			if len(args.args) == 0 {
				return strObject(""), nil
			}

			var line strings.Builder
			for _, arg := range args.args {
				if s, ok := arg.(fmt.Stringer); ok {
					line.WriteString(s.String())
				}
			}

			return strObject(line.String()), nil
		}))

		i.rootEC.addCmd("pipe", invokableFunc(func(ctx context.Context, args invocationArgs) (object, error) {
			return &listIterStream{
				list: args.args,
			}, nil
		}))

		i.rootEC.addCmd("joinpipe", invokableStreamFunc(func(ctx context.Context, inStream stream, args invocationArgs) (object, error) {
			sb := strings.Builder{}
			if err := forEach(inStream, func(o object, i int) error {
				if i > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(o.String())
				return nil
			}); err != nil {
				return nil, err
			}
			return strObject(sb.String()), nil
		}))

		i.rootEC.setVar("a", strObject("alpha"))
		i.rootEC.setVar("bee", strObject("buzz"))
	}
}

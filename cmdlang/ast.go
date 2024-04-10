package cmdlang

import (
	"github.com/alecthomas/participle/v2"
	"io"
)

type astLiteral struct {
	Str   *string `parser:"@String"`
	Ident *string `parser:" | @Ident"`
}

type astCmdArg struct {
	Literal *astLiteral  `parser:"@@"`
	Sub     *astPipeline `parser:"| '(' @@ ')'"`
}

type astCmd struct {
	Name string      `parser:"@Ident"`
	Args []astCmdArg `parser:"@@*"`
}

type astPipeline struct {
	First *astCmd   `parser:"@@"`
	Rest  []*astCmd `parser:"( '|' @@ )*"`
}

var parser = participle.MustBuild[astPipeline]()

func parse(r io.Reader) (*astPipeline, error) {
	return parser.Parse("test", r)
}

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
	Var     *string      `parser:"| '$' @Ident"`
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

type astStatements struct {
	First *astPipeline   `parser:"@@"`
	Rest  []*astPipeline `parser:"( ';' @@ )*"` // TODO: also add support for newlines
}

var parser = participle.MustBuild[astStatements]()

func parse(r io.Reader) (*astStatements, error) {
	return parser.Parse("test", r)
}

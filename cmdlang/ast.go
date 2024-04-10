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
	Literal astLiteral `parser:"@@"`
}

type astCmd struct {
	Name string      `parser:"@Ident"`
	Args []astCmdArg `parser:"@@*"`
}

var parser = participle.MustBuild[astCmd]()

func parse(r io.Reader) (*astCmd, error) {
	return parser.Parse("test", r)
}

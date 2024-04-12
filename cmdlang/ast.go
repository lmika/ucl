package cmdlang

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"io"
)

type astLiteral struct {
	Str   *string `parser:"@String"`
	Ident *string `parser:" | @Ident"`
}

type astCmdArg struct {
	Literal *astLiteral  `parser:"@@"`
	Var     *string      `parser:"| DOLLAR @Ident"`
	Sub     *astPipeline `parser:"| LP @@ RP"`
}

type astCmd struct {
	Name string      `parser:"@Ident"`
	Args []astCmdArg `parser:"@@*"`
}

type astPipeline struct {
	First *astCmd   `parser:"@@"`
	Rest  []*astCmd `parser:"( PIPE @@ )*"`
}

type astStatements struct {
	First *astPipeline   `parser:"@@"`
	Rest  []*astPipeline `parser:"( (SEMICL | NL)+ @@ )*"` // TODO: also add support for newlines
}

type astBlock struct {
	Statements *astStatements `parser:"'{'  "`
}

var scanner = lexer.MustStateful(lexer.Rules{
	"Root": {
		{"Whitespace", `[ ]`, nil},
		{"NL", `\n\s*`, nil},
		{"String", `"(\\"|[^"])*"`, nil},
		{"DOLLAR", `\$`, nil},
		{"LP", `\(`, nil},
		{"RP", `\)`, nil},
		{"SEMICL", `;`, nil},
		{"PIPE", `\|`, nil},
		{"Ident", `\w+`, nil},
	},
})
var parser = participle.MustBuild[astStatements](participle.Lexer(scanner),
	participle.Elide("Whitespace"))

func parse(r io.Reader) (*astStatements, error) {
	return parser.Parse("test", r)
}

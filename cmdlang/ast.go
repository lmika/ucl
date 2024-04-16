package cmdlang

import (
	"io"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type astLiteral struct {
	Str *string `parser:"@String"`
}

type astHashKey struct {
	Literal *astLiteral  `parser:"@@"`
	Ident   *string      `parser:"| @Ident"`
	Var     *string      `parser:"| DOLLAR @Ident"`
	Sub     *astPipeline `parser:"| LP @@ RP"`
}

type astElementPair struct {
	Left  astCmdArg  `parser:"@@"`
	Right *astCmdArg `parser:"( COLON @@ )? NL?"`
}

type astListOrHash struct {
	EmptyList bool              `parser:"@(LS RS)"`
	EmptyHash bool              `parser:"| @(LS COLON RS)"`
	Elements  []*astElementPair `parser:"| LS NL? @@+ @@* RS"`
}

type astBlock struct {
	Statements []*astStatements `parser:"LC NL? @@ NL? RC"`
}

type astCmdArg struct {
	Literal    *astLiteral    `parser:"@@"`
	Ident      *string        `parser:"| @Ident"`
	Var        *string        `parser:"| DOLLAR @Ident"`
	Sub        *astPipeline   `parser:"| LP @@ RP"`
	ListOrHash *astListOrHash `parser:"| @@"`
	Block      *astBlock      `parser:"| @@"`
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
	Rest  []*astPipeline `parser:"( NL+ @@ )*"` // TODO: also add support for newlines
}

type astScript struct {
	Statements *astStatements `parser:"NL* @@ NL*"`
}

var scanner = lexer.MustStateful(lexer.Rules{
	"Root": {
		{"Whitespace", `[ \t]+`, nil},
		{"String", `"(\\"|[^"])*"`, nil},
		{"DOLLAR", `\$`, nil},
		{"COLON", `\:`, nil},
		{"LP", `\(`, nil},
		{"RP", `\)`, nil},
		{"LS", `\[`, nil},
		{"RS", `\]`, nil},
		{"LC", `\{`, nil},
		{"RC", `\}`, nil},
		{"NL", `[;\n][; \n\t]*`, nil},
		{"PIPE", `\|`, nil},
		{"Ident", `[\w-]+`, nil},
	},
})
var parser = participle.MustBuild[astScript](participle.Lexer(scanner),
	participle.Elide("Whitespace"))

func parse(r io.Reader) (*astScript, error) {
	return parser.Parse("test", r)
}

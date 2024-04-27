package ucl

import (
	"io"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type astLiteral struct {
	Str *string `parser:"@String"`
	Int *int    `parser:"| @Int"`
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
	Names      []string         `parser:"LC NL? (PIPE @Ident+ PIPE NL?)?"`
	Statements []*astStatements `parser:"@@ NL? RC"`
}

type astMaybeSub struct {
	Sub *astPipeline `parser:"@@?"`
}

type astCmdArg struct {
	Literal    *astLiteral    `parser:"@@"`
	Ident      *string        `parser:"| @Ident"`
	Var        *string        `parser:"| DOLLAR @Ident"`
	MaybeSub   *astMaybeSub   `parser:"| LP @@ RP"`
	ListOrHash *astListOrHash `parser:"| @@"`
	Block      *astBlock      `parser:"| @@"`
}

type astCmd struct {
	Name astCmdArg   `parser:"@@"`
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
	Statements *astStatements `parser:"NL* (@@ NL*)?"`
}

var scanner = lexer.MustStateful(lexer.Rules{
	"Root": {
		{"Whitespace", `[ \t]+`, nil},
		{"Comment", `[#].*`, nil},
		{"String", `"(\\"|[^"])*"`, nil},
		{"Int", `[-]?[0-9][0-9]*`, nil},
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
		{"Ident", `[-]*[a-zA-Z_][\w-]*`, nil},
	},
})
var parser = participle.MustBuild[astScript](participle.Lexer(scanner),
	participle.Elide("Whitespace", "Comment"))

func parse(r io.Reader) (*astScript, error) {
	return parser.Parse("test", r)
}

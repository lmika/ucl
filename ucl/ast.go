package ucl

import (
	"io"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type astLiteral struct {
	Str *string `parser:"@String"`
	Int *int    `parser:"| @Int"`
}

type astIdentNames struct {
	Ident      string   `parser:"@Ident"`
	ColonParts []string `parser:"( COLON @Ident )*"`
}

func (ai *astIdentNames) String() string {
	sb := strings.Builder{}
	sb.WriteString(ai.Ident)
	for _, p := range ai.ColonParts {
		sb.WriteRune(':')
		sb.WriteString(p)
	}
	return sb.String()
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
	Ident      *astIdentNames `parser:"| @@"`
	Var        *string        `parser:"| DOLLAR @Ident"`
	MaybeSub   *astMaybeSub   `parser:"| LP @@ RP"`
	ListOrHash *astListOrHash `parser:"| @@"`
	Block      *astBlock      `parser:"| @@"`
}

type astDotSuffix struct {
	KeyIdent *astIdentNames `parser:"@@"`
	Pipeline *astPipeline   `parser:"| LP @@ RP"`
}

type astDot struct {
	Arg       astCmdArg      `parser:"@@"`
	DotSuffix []astDotSuffix `parser:"( DOT @@ )*"`
}

type astCmd struct {
	Name astDot   `parser:"@@"`
	Args []astDot `parser:"@@*"`
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
		{"DOT", `[.]`, nil},
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

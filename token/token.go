package token

const (
	ILLEGAL = "ILLEGAL" // ILLEGAL signifies a token/character we don’t know about
	EOF     = "EOF"     // EOF stands for "end of file", which tells our parser later on that it can stop

	// identifiers + literals
	IDENT = "IDENT" // 	add, x, foo, ...
	INT   = "INT"   // 12345

	// operators
	ASSIGN = "ASSIGN"
	PLUS   = "PLUS"

	// Delimeters
	COMMA     = ","
	SEMICOLON = ";"

	LPAREN = "("
	RPAREN = ")"
	LBRACE = "{"
	RBRACE = "}"

	// Keywords
	FUNCTION = "FUNCTION"
	LET      = "LET"
)

var keywords = map[string]TokenType{
	"fn":  FUNCTION,
	"let": LET,
}

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}

	return IDENT

}

package token

const (
	ILLEGAL = "ILLEGAL" // ILLEGAL signifies a token/character we don’t know about
	EOF     = "EOF"     // EOF stands for "end of file", which tells our parser later on that it can stop

	// identifiers + literals
	IDENT = "IDENT" // 	add, x, foo, ...
	INT   = "INT"   // 12345

	STRING = "STRING"

	// operators
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"
	LT       = "<"
	GT       = ">"

	// Delimeters
	COMMA     = ","
	SEMICOLON = ";"

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// Keywords
	FUNCTION = "FUNCTION"
	LET      = "LET"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"

	EQ     = "=="
	NOT_EQ = "!="

	COLON = ":"
)

var keywords = map[string]TokenType{
	"fn":     FUNCTION,
	"let":    LET,
	"true":   TRUE,
	"false":  FALSE,
	"if":     IF,
	"else":   ELSE,
	"return": RETURN,
}

type TokenType string

/*
*
Token representation of  "let x = 5 + 5;"
[

	LET,
	IDENTIFIER("x"),
	EQUAL_SIGN,
	INTEGER(5),
	PLUS_SIGN,
	INTEGER(5),
	SEMICOLON

]
*/
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

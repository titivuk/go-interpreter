package parser

import (
	"fmt"
	"strconv"

	"github.com/titivuk/go-interpreter/ast"
	"github.com/titivuk/go-interpreter/lexer"
	"github.com/titivuk/go-interpreter/token"
)

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression // param is left side of infix operator
)

const (
	_ int = iota // use iota to give the following constants incrementing numbers as values
	LOWEST
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
	INDEX
)

var precedences = map[token.TokenType]int{
	token.EQ:       EQUALS,
	token.NOT_EQ:   EQUALS,
	token.LT:       LESSGREATER,
	token.GT:       LESSGREATER,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.ASTERISK: PRODUCT,
	token.SLASH:    PRODUCT,
	token.LPAREN:   CALL,
	token.LBRACKET: INDEX,
}

type Parser struct {
	l *lexer.Lexer

	currToken token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn

	errors []string
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefixFn(token.IDENT, p.parseIdentifier)
	p.registerPrefixFn(token.INT, p.parseIntegerLiteral)
	p.registerPrefixFn(token.STRING, p.parseStringLiteral)
	p.registerPrefixFn(token.BANG, p.parsePrefixExpression)
	p.registerPrefixFn(token.MINUS, p.parsePrefixExpression)
	p.registerPrefixFn(token.TRUE, p.parseBoolean)
	p.registerPrefixFn(token.FALSE, p.parseBoolean)
	p.registerPrefixFn(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefixFn(token.IF, p.parseIfExpression)
	p.registerPrefixFn(token.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefixFn(token.LBRACKET, p.parseArrayLiteral)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfixFn(token.EQ, p.parseInfixExpression)
	p.registerInfixFn(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfixFn(token.LT, p.parseInfixExpression)
	p.registerInfixFn(token.GT, p.parseInfixExpression)
	p.registerInfixFn(token.PLUS, p.parseInfixExpression)
	p.registerInfixFn(token.MINUS, p.parseInfixExpression)
	p.registerInfixFn(token.ASTERISK, p.parseInfixExpression)
	p.registerInfixFn(token.SLASH, p.parseInfixExpression)
	p.registerInfixFn(token.LPAREN, p.parseCallExpression)
	p.registerInfixFn(token.LBRACKET, p.parseIndexExpression)

	// Read two tokens, so currToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	// parse until we reach the end
	for !p.currTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead",
		t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) registerPrefixFn(tokenTpye token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenTpye] = fn
}

func (p *Parser) registerInfixFn(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) nextToken() {
	p.currToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.currToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseLetStatement() ast.Statement {
	stmt := &ast.LetStatement{Token: p.currToken}

	// next token must be IDENT
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.currToken, Value: p.currToken.Literal}

	// next token must be ASSIGN
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseReturnStatement() ast.Statement {
	stmt := &ast.ReturnStatement{Token: p.currToken}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.currToken}

	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt

}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.currToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.currToken.Type)
		return nil
	}

	expression := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		// if there is no infix parser => it's prefix expression => return immediately
		if infix == nil {
			return expression
		}

		p.nextToken()

		// since we've checked that infix exist
		// so currToken is infix operator and expression is left expression
		// and we call infix parse function
		expression = infix(expression)
	}

	return expression
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.currToken, Value: p.currToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	value, err := strconv.ParseInt(p.currToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.currToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	return &ast.IntegerLiteral{Token: p.currToken, Value: value}
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.currToken, Value: p.currToken.Literal}
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.currToken, Value: p.currTokenIs(token.TRUE)}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{Token: p.currToken, Operator: p.currToken.Literal}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.currToken,
		Operator: p.currToken.Literal,
		Left:     left,
	}

	precedence := p.currPrecedence() // curr is infix operator
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	// p.parseExpression call above is going to stop when peekToken = RPAREN
	// because p.peekPrecedence returns 0 for RPAREN, so for loop stops inside the parseExpression fn
	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.currToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Consequence = p.parseBlockStatement()

	// parse else block if there is token.ELSE token
	if p.peekTokenIs(token.ELSE) {
		// advance the token, now it points to token.LBRACE
		p.nextToken()

		// assert nextToken is token.LBRACE
		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		expression.Alternative = p.parseBlockStatement()
	}

	return expression
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.currToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.currTokenIs(token.RBRACE) && !p.currTokenIs(token.EOF) {
		statement := p.parseStatement()
		if statement != nil {
			block.Statements = append(block.Statements, statement)
		}

		p.nextToken()

	}

	return block
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	expression := &ast.FunctionLiteral{Token: p.currToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	expression.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Body = p.parseBlockStatement()

	return expression
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	parameters := []*ast.Identifier{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return parameters
	}

	p.nextToken()

	ident := &ast.Identifier{Token: p.currToken, Value: p.currToken.Literal}
	parameters = append(parameters, ident)
	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		ident := &ast.Identifier{Token: p.currToken, Value: p.currToken.Literal}
		parameters = append(parameters, ident)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return parameters
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	expression := &ast.CallExpression{Token: p.currToken, Function: function}
	expression.Arguments = p.parseCallArguments()

	return expression
}

func (p *Parser) parseCallArguments() []ast.Expression {
	arguments := []ast.Expression{}
	arguments = append(arguments, p.parseExpressionList(token.RPAREN)...)

	return arguments
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	arrayExpression := &ast.ArrayLiteral{Token: p.currToken}
	arrayExpression.Elements = p.parseExpressionList(token.RBRACKET)

	return arrayExpression
}

func (p *Parser) parseExpressionList(endToken token.TokenType) []ast.Expression {
	expressions := []ast.Expression{}

	if p.peekTokenIs(endToken) {
		p.nextToken()
		return expressions
	}

	// cur token points to the start of the first element after this call
	p.nextToken()

	// parse first element of the list
	expressions = append(expressions, p.parseExpression(LOWEST))

	// loop ends when curToken points to end pos of the last element in the list
	for p.peekTokenIs(token.COMMA) {
		// advance pos twice because p.parseExpression called above leaves p.currToken on last pos related to the expression
		p.nextToken() // sets curToken to ','
		p.nextToken() // sets curToken to first char of the next element

		expressions = append(expressions, p.parseExpression(LOWEST))
	}

	// since currToken points to the last pos of the last element in the array
	// next token must be endToken
	if !p.expectPeek(endToken) {
		return nil
	}

	return expressions
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	indexExpression := &ast.IndexExpression{
		Token: p.currToken,
		Left:  left,
	}

	p.nextToken()
	indexExpression.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return indexExpression
}

func (p *Parser) currTokenIs(t token.TokenType) bool {
	return p.currToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

// checks the type of the peekToken and only if the type is correct does it advance the tokens by calling nextToken
func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}

	p.peekError(t)
	return false
}

// returns the precedence associated with the token type of p.peekToken
func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}

	return LOWEST
}

// returns the precedence associated with the token type of p.currToken
func (p *Parser) currPrecedence() int {
	if p, ok := precedences[p.currToken.Type]; ok {
		return p
	}

	return LOWEST
}

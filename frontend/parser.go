package frontend

import (
	"fmt"

	"github.com/isaacev/Plaid/feedback"
	"github.com/isaacev/Plaid/source"
)

// Parse takes a file and returns an abstract-syntax-tree and any errors/warnings
// generated during the parsing process
func Parse(file *source.File) (ast *Program, msgs []feedback.Message) {
	var msg feedback.Message

	parser := NewParser(file)
	ast, msg = parser.Parse()

	if msg != nil {
		msgs = append(msgs, msg)
	}

	return ast, msgs
}

type binaryParselet func(*Parser, Token, Node) (Node, feedback.Message)
type unaryParselet func(*Parser, Token) (Node, feedback.Message)

// Parser instances contain a Lexer instance and tables of unary and binary
// operator precedences and parselets
type Parser struct {
	Lexer            *Lexer
	binaryPrecedence map[TokenSymbol]int
	unaryPrecedence  map[TokenSymbol]int
	binaryParselets  map[TokenSymbol]binaryParselet
	unaryParselets   map[TokenSymbol]unaryParselet
}

// NewParser is a Parser factory function that populates the Parser's parselet
// table with the appropriate symbols, precedence values and parselet functions
// func NewParser(lexer *Lexer) *Parser {
func NewParser(file *source.File) *Parser {
	grammar := &Grammar{
		OperatorRunes:   []rune{'+', '-', '*', '/', ':', '=', '<', '>'},
		PunctuatorRunes: []rune{'(', ')', '{', '}', ';', ',', '#'},
		Keywords:        []string{"let", "print", "return", "if", "else", "loop", "end"},
	}

	lexer := NewLexer(file, grammar)

	p := &Parser{
		Lexer:            lexer,
		binaryPrecedence: make(map[TokenSymbol]int),
		unaryPrecedence:  make(map[TokenSymbol]int),
		binaryParselets:  make(map[TokenSymbol]binaryParselet),
		unaryParselets:   make(map[TokenSymbol]unaryParselet),
	}

	p.addUnaryParselet(IntegerSymbol, 0, literalParselet)
	p.addUnaryParselet(DecimalSymbol, 0, literalParselet)
	p.addUnaryParselet(StringSymbol, 0, literalParselet)
	p.addUnaryParselet(IdentSymbol, 0, identParselet)

	p.addUnaryParselet(TokenSymbol("("), 0, leadingParenParselet)
	p.addBinaryParselet(TokenSymbol("("), 101, dispatchParselet)

	p.addBinaryParselet(TokenSymbol(":="), 10, assignmentParselet)
	p.addBinaryParselet(TokenSymbol("::"), 110, typeAssociationParselet)

	p.addBinaryParselet(TokenSymbol("<"), 40, binaryInfixParselet(40))
	p.addBinaryParselet(TokenSymbol(">"), 40, binaryInfixParselet(40))
	p.addBinaryParselet(TokenSymbol("<="), 40, binaryInfixParselet(40))
	p.addBinaryParselet(TokenSymbol(">="), 40, binaryInfixParselet(40))
	p.addBinaryParselet(TokenSymbol("=="), 40, binaryInfixParselet(40))

	p.addBinaryParselet(TokenSymbol("+"), 50, binaryInfixParselet(50))
	p.addBinaryParselet(TokenSymbol("-"), 50, binaryInfixParselet(50))
	p.addBinaryParselet(TokenSymbol("*"), 60, binaryInfixParselet(60))
	p.addBinaryParselet(TokenSymbol("/"), 60, binaryInfixParselet(60))

	p.addBinaryParselet(TokenSymbol("=>"), 100, functionParselet)

	p.addUnaryParselet(TokenSymbol("let"), 100, letDeclarationParselet)

	p.addUnaryParselet(TokenSymbol("print"), 110, printStatementParselet)
	p.addUnaryParselet(TokenSymbol("return"), 110, returnStatementParselet)
	p.addUnaryParselet(TokenSymbol("if"), 110, ifStatementParselet)
	p.addUnaryParselet(TokenSymbol("loop"), 110, loopStatementParselet)

	return p
}

func (p *Parser) addBinaryParselet(sym TokenSymbol, precedence int, parselet binaryParselet) {
	p.binaryPrecedence[sym] = precedence
	p.binaryParselets[sym] = parselet
}

func (p *Parser) addUnaryParselet(sym TokenSymbol, precedence int, parselet unaryParselet) {
	p.unaryPrecedence[sym] = precedence
	p.unaryParselets[sym] = parselet
}

func (p *Parser) nextPrecedence() (prec int, msg feedback.Message) {
	tok, msg := p.Lexer.Peek()

	if msg == nil {
		if prec, ok := p.binaryPrecedence[tok.Symbol]; ok {
			return prec, nil
		}

		// emit an error if the upcoming symbol is NOT one of the
		// following token symbols
		if tok.Symbol != TokenSymbol(";") &&
			tok.Symbol != TokenSymbol(":") &&
			tok.Symbol != TokenSymbol(",") &&
			tok.Symbol != TokenSymbol(")") &&
			tok.Symbol != TokenSymbol("}") &&
			tok.Symbol != TokenSymbol("]") {
			return 0, feedback.Error{
				Classification: feedback.SyntaxError,
				File:           p.Lexer.Scanner.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("Unexpected `%s`", tok.Symbol),
					Span:        tok.Span,
				},
			}
		}
	}

	return 0, msg
}

// parseExpression returns a node representing the next expression so long as
// the next expression does not have less precedence than the "precedence"
// parameter
func (p *Parser) parseExpression(precedence int) (node Node, msg feedback.Message) {
	var tok Token

	if tok, msg = p.Lexer.Next(); msg != nil {
		return nil, msg
	} else if tok.Symbol == EOFSymbol {
		return nil, feedback.Error{
			Classification: feedback.SyntaxError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: "Unexpected end of program",
				Span:        tok.Span,
			},
		}
	}

	if unaryParselet, ok := p.unaryParselets[tok.Symbol]; ok {
		var leftNode Node

		if leftNode, msg = unaryParselet(p, tok); msg != nil {
			return nil, msg
		}

		// left-associated expressions based on their relative precedence
		for {
			var nextPrecedence int

			// catch syntax errors produced by checking the precedence of the next token
			if nextPrecedence, msg = p.nextPrecedence(); msg != nil {
				return nil, msg
			} else if precedence >= nextPrecedence {
				break
			}

			if tok, msg = p.Lexer.Next(); msg != nil {
				// next token produced an error
				return nil, msg
			}

			if infixParselet, ok := p.binaryParselets[tok.Symbol]; ok {
				if leftNode, msg = infixParselet(p, tok, leftNode); msg != nil {
					return nil, msg
				}
			} else {
				return nil, feedback.Error{
					Classification: feedback.SyntaxError,
					File:           p.Lexer.Scanner.File,
					What: feedback.Selection{
						Description: fmt.Sprintf("Unexpected `%s`", tok.Symbol),
						Span:        tok.Span,
					},
				}
			}
		}

		return leftNode, nil
	}

	return nil, feedback.Error{
		Classification: feedback.SyntaxError,
		File:           p.Lexer.Scanner.File,
		What: feedback.Selection{
			Description: fmt.Sprintf("Unexpected `%s`", tok.Symbol),
			Span:        tok.Span,
		},
	}
}

// parseStatementsUntil collects statements until it encounters a token matching
// the given "terminator" parameter. This function is used to parse the bodies
// of functions and conditional statements
func (p *Parser) parseStatementsUntil(terminator TokenSymbol) (stmts []Stmt, msg feedback.Message) {
	for {
		var tok Token
		var node Node

		if node, msg = p.parseExpression(0); msg != nil {
			return stmts, msg
		}

		if stmt := node.(Stmt); stmt != nil {
			stmts = append(stmts, stmt)
		} else {
			return stmts, feedback.Error{
				Classification: feedback.SyntaxError,
				File:           p.Lexer.Scanner.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("%T is not a statement", node),
					Span:        source.Span{Start: node.Pos(), End: node.End()},
				},
			}
		}

		// expect semicolon to follow each expression in block
		if tok, msg = p.Lexer.ExpectNext(TokenSymbol(";")); msg != nil {
			return stmts, feedback.Error{
				Classification: feedback.SyntaxError,
				File:           p.Lexer.Scanner.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("Expected semicolon or newline after expression, instead found '%s'", tok.Symbol),
					Span:        tok.Span,
				},
			}
		}

		if p.Lexer.PeekMatches(terminator) {
			return stmts, nil
		}
	}
}

// parseFunctionBody returns a FunctionBody struct representing the collection
// of statements between braces in a function body
func (p *Parser) parseFunctionBody() (body *FunctionBody, msg feedback.Message) {
	var lBrace Token
	var stmts []Stmt
	var rBrace Token

	if lBrace, msg = p.Lexer.ExpectNext(TokenSymbol("{")); msg != nil {
		return nil, msg
	}

	if stmts, msg = p.parseStatementsUntil(TokenSymbol("}")); msg != nil {
		return nil, msg
	}

	if rBrace, msg = p.Lexer.ExpectNext(TokenSymbol("}")); msg != nil {
		return nil, msg
	}

	return &FunctionBody{
		Statements: stmts,
		LeftBrace:  lBrace,
		RightBrace: rBrace,
	}, nil
}

// parseConditionalBody returns a ConditionalBody struct representing the
// statements between a colon and an "end" keyword that make up the body of a
// conditional statement
func (p *Parser) parseConditionalBody() (body *ConditionalBody, msg feedback.Message) {
	var colon Token
	var stmts []Stmt
	var endKeyword Token

	if colon, msg = p.Lexer.ExpectNext(TokenSymbol(":")); msg != nil {
		return nil, msg
	}

	if stmts, msg = p.parseStatementsUntil(TokenSymbol("end")); msg != nil {
		return nil, msg
	}

	if endKeyword, msg = p.Lexer.ExpectNext(TokenSymbol("end")); msg != nil {
		return nil, msg
	}

	return &ConditionalBody{
		Statements: stmts,
		Colon:      colon,
		EndKeyword: endKeyword,
	}, nil
}

// Parse produces an AST from a set of parselets, a grammar and a lexer
func (p *Parser) Parse() (node *Program, msg feedback.Message) {
	stmts, msg := p.parseStatementsUntil(EOFSymbol)

	return &Program{
		Statements: stmts,
	}, msg
}

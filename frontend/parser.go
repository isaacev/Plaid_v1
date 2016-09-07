package frontend

import (
	"fmt"

	"github.com/isaacev/Plaid/feedback"
	"github.com/isaacev/Plaid/source"
)

// Parse takes a file and returns an abstract-syntax-tree and any errors/warnings
// generated during the parsing process
func Parse(file *source.File) (ast *ProgramNode, msgs []feedback.Message) {
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
		PunctuatorRunes: []rune{'[', ']', '(', ')', '{', '}', ';', ',', '#'},
		Keywords: []string{
			"fn",
			"let",
			"print",
			"return",
			"if",
			"elif",
			"else",
			"loop",
			"end",
		},
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
	p.addUnaryParselet(TokenSymbol("fn"), 0, funcParselet)
	p.addUnaryParselet(LBracketSymbol, 0, listParselet)

	p.addUnaryParselet(LParenSymbol, 0, groupParselet)
	p.addBinaryParselet(LParenSymbol, 80, dispatchParselet)

	// TODO: should assignment be kept an expression?
	p.addBinaryParselet(TokenSymbol(":="), 10, assignmentParselet)

	// Logical comparison expressions
	p.addBinaryParselet(TokenSymbol("<"), 40, binaryInfixParselet(40))
	p.addBinaryParselet(TokenSymbol(">"), 40, binaryInfixParselet(40))
	p.addBinaryParselet(TokenSymbol("<="), 40, binaryInfixParselet(40))
	p.addBinaryParselet(TokenSymbol(">="), 40, binaryInfixParselet(40))
	p.addBinaryParselet(TokenSymbol("=="), 40, binaryInfixParselet(40))

	// Arithmetic expressions
	p.addBinaryParselet(TokenSymbol("+"), 50, binaryInfixParselet(50))
	p.addBinaryParselet(TokenSymbol("-"), 50, binaryInfixParselet(50))
	p.addBinaryParselet(TokenSymbol("*"), 60, binaryInfixParselet(60))
	p.addBinaryParselet(TokenSymbol("/"), 60, binaryInfixParselet(60))

	// List index access
	p.addBinaryParselet(LBracketSymbol, 80, indexAccessParselet)

	// All statements have a binding-power of 0
	p.addUnaryParselet(TokenSymbol("let"), 0, letDeclarationParselet)
	p.addUnaryParselet(TokenSymbol("return"), 0, returnStatementParselet)
	p.addUnaryParselet(TokenSymbol("if"), 0, ifStatementParselet)
	p.addUnaryParselet(TokenSymbol("loop"), 0, loopStatementParselet)
	p.addUnaryParselet(TokenSymbol("print"), 0, printStatementParselet)

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
// the given "terminatorMatches" function. This function is used to parse the
// bodies of functions and conditional statements
func (p *Parser) parseStatementsUntil(terminatorMatches func(Token) bool) (stmts []Stmt, msg feedback.Message) {
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

		if tok, msg := p.Lexer.Peek(); msg != nil {
			return nil, msg
		} else if terminatorMatches(tok) {
			return stmts, nil
		}
	}
}

// Parse produces an AST from a set of parselets, a grammar and a lexer
func (p *Parser) Parse() (node *ProgramNode, msg feedback.Message) {
	stmts, msg := p.parseStatementsUntil(func(tok Token) bool { return tok.Symbol == EOFSymbol })

	return &ProgramNode{
		Statements: stmts,
	}, msg
}

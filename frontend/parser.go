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

type postfixParselet func(*Parser, Token, Node) (Node, feedback.Message)
type prefixParselet func(*Parser, Token) (Node, feedback.Message)

// Parser instances contain a Lexer instance and tables of prefix and postfix
// operator precedences and parselets
type Parser struct {
	Lexer            *Lexer
	postfixPrecedence map[TokenSymbol]int
	prefixPrecedence  map[TokenSymbol]int
	postfixParselets  map[TokenSymbol]postfixParselet
	prefixParselets   map[TokenSymbol]prefixParselet
}

// NewParser is a Parser factory function that populates the Parser's parselet
// table with the appropriate symbols, precedence values and parselet functions
// func NewParser(lexer *Lexer) *Parser {
func NewParser(file *source.File) *Parser {
	grammar := &Grammar{
		OperatorRunes:   []rune{
			'+',
			'-',
			'*',
			'/',
			':',
			'=',
			'<',
			'>',
			'?',
			'!',
		},
		PunctuatorRunes: []rune{
			'[',
			']',
			'(',
			')',
			'{',
			'}',
			';',
			',',
			'#',
		},
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
		postfixPrecedence: make(map[TokenSymbol]int),
		prefixPrecedence:  make(map[TokenSymbol]int),
		postfixParselets:  make(map[TokenSymbol]postfixParselet),
		prefixParselets:   make(map[TokenSymbol]prefixParselet),
	}

	p.addPrefixParselet(BooleanSymbol, 0, literalParselet)
	p.addPrefixParselet(IntegerSymbol, 0, literalParselet)
	p.addPrefixParselet(DecimalSymbol, 0, literalParselet)
	p.addPrefixParselet(StringSymbol, 0, literalParselet)
	p.addPrefixParselet(IdentSymbol, 0, identParselet)
	p.addPrefixParselet(TokenSymbol("fn"), 0, funcParselet)
	p.addPrefixParselet(LBracketSymbol, 0, listParselet)

	p.addPrefixParselet(LParenSymbol, 0, groupParselet)
	p.addPostfixParselet(LParenSymbol, 80, dispatchParselet)

	// TODO: should assignment be kept an expression?
	p.addPostfixParselet(TokenSymbol(":="), 10, assignmentParselet)

	// Logical comparison expressions
	p.addPostfixParselet(TokenSymbol("<"), 40, binaryInfixParselet(40))
	p.addPostfixParselet(TokenSymbol(">"), 40, binaryInfixParselet(40))
	p.addPostfixParselet(TokenSymbol("<="), 40, binaryInfixParselet(40))
	p.addPostfixParselet(TokenSymbol(">="), 40, binaryInfixParselet(40))
	p.addPostfixParselet(TokenSymbol("=="), 40, binaryInfixParselet(40))

	// Arithmetic expressions
	p.addPostfixParselet(TokenSymbol("++"), 50, binaryInfixParselet(50))
	p.addPostfixParselet(TokenSymbol("+"), 50, binaryInfixParselet(50))
	p.addPostfixParselet(TokenSymbol("-"), 50, binaryInfixParselet(50))
	p.addPostfixParselet(TokenSymbol("*"), 60, binaryInfixParselet(60))
	p.addPostfixParselet(TokenSymbol("/"), 60, binaryInfixParselet(60))
	p.addPrefixParselet(TokenSymbol("-"), 70, unaryPrefixParselet(70))


	// List index access
	p.addPostfixParselet(LBracketSymbol, 80, indexAccessParselet)

	// All statements have a binding-power of 0
	p.addPrefixParselet(TokenSymbol("let"), 0, letDeclarationParselet)
	p.addPrefixParselet(TokenSymbol("return"), 0, returnStatementParselet)
	p.addPrefixParselet(TokenSymbol("if"), 0, ifStatementParselet)
	p.addPrefixParselet(TokenSymbol("loop"), 0, loopStatementParselet)
	p.addPrefixParselet(TokenSymbol("print"), 0, printStatementParselet)

	return p
}

func (p *Parser) addPostfixParselet(sym TokenSymbol, precedence int, parselet postfixParselet) {
	p.postfixPrecedence[sym] = precedence
	p.postfixParselets[sym] = parselet
}

func (p *Parser) addPrefixParselet(sym TokenSymbol, precedence int, parselet prefixParselet) {
	p.prefixPrecedence[sym] = precedence
	p.prefixParselets[sym] = parselet
}

func (p *Parser) nextPrecedence() (prec int, msg feedback.Message) {
	tok, msg := p.Lexer.Peek()

	if msg == nil {
		if prec, ok := p.postfixPrecedence[tok.Symbol]; ok {
			return prec, nil
		}

		// emit an error if the upcoming symbol is NOT one of the
		// following token symbols
		if tok.Symbol != TokenSymbol(";") &&
			tok.Symbol != TokenSymbol(":") &&
			tok.Symbol != TokenSymbol(",") &&
			tok.Symbol != TokenSymbol(")") &&
			tok.Symbol != TokenSymbol("}") &&
			tok.Symbol != RInterpolSymbol &&
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

	if prefixParselet, ok := p.prefixParselets[tok.Symbol]; ok {
		var leftNode Node

		if leftNode, msg = prefixParselet(p, tok); msg != nil {
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

			if infixParselet, ok := p.postfixParselets[tok.Symbol]; ok {
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

package frontend

import (
	"fmt"

	"github.com/isaacev/Plaid/feedback"
	"github.com/isaacev/Plaid/source"
)

// Lexer structs maintain state during the lexical analysis of a chunk of source
// code, generating a sequence of Tokens. As part of the Plaid frontend, Lexers
// are also responsible for semicolon insertion
type Lexer struct {
	Scanner        *Scanner
	Grammar        *Grammar
	peekBuffer     []Token
	msgBuffer      []feedback.Message
	histBuffer     []Token
	eof            bool
	openParenStack []int
	openStrings    int

	forceStringLexing bool
}

// NewLexer is a constructor function that takes a Scanner and a Grammar and
// returns a reference to a newly minted Lexer struct
func NewLexer(file *source.File, grammar *Grammar) *Lexer {
	scanner := NewScanner(file)

	return &Lexer{
		Scanner:        scanner,
		Grammar:        grammar,
		peekBuffer:     []Token{},
		msgBuffer:      []feedback.Message{},
		histBuffer:     []Token{},
		eof:            false,
		openParenStack: []int{0},
		openStrings:    0,
		forceStringLexing: false,
	}
}

func (l *Lexer) deferToken(tok Token, msg feedback.Message) {
	l.peekBuffer = append(l.peekBuffer, tok)
	l.msgBuffer = append(l.msgBuffer, msg)
}

// readNextToken is responsible for digesting characters from a scanner and
// producing the next Token. This function advances the scanner and is only
// called when the peekBuffer is totally exhausted.
func (l *Lexer) readNextToken() (tok Token, msg feedback.Message) {
	// If the lexer has already emitted an EOF token, the `l.eof` flag will be
	// true meaning the scanner has been exhausted and any subsequent hits will
	// panic. To avoid this, just emit an EOF token immediately
	if l.eof {
		pos := source.Pos{l.Scanner.nextLine, l.Scanner.nextCol}
		span := source.Span{pos, pos}
		eofToken := Token{EOFSymbol, "<EOF>", span}

		if len(l.histBuffer) > 0 {
			// Isolate the last token emitted
			lastTok := l.histBuffer[len(l.histBuffer)-1]

			// If some tokens have been emitted, set the EOF token's span to the
			// last token so that error messages will point to the last
			// meaningful syntax token and not some empty line at the end of the
			// file
			eofToken.Span = lastTok.Span

			// If the last token emitted was an EOF token, just emit another
			if lastTok.Symbol == EOFSymbol {
				return eofToken, nil
			}

			// Check if a semicolon should be inserted between the last
			// statement and the EOF
			if l.Grammar.canInsertSemicolonAfter(lastTok) {
				// Make a synthetic semicolon token and gives it a position 1
				// column beyond the end of the last token in the token history
				// buffer
				return Token{TokenSymbol(";"), ";", span}, nil
			}
		}

		// If the file is empty of semantic syntax, just emit an EOF token
		return eofToken, nil
	}

	peek, _, _, _ := l.Scanner.Peek()

	if l.forceStringLexing {
		return l.lexString()
	}

	if l.Grammar.isWhitespace(peek) {
		return l.lexWhitespace()
	} else if l.Grammar.isCommentStart(peek) {
		return l.lexComment()
	} else if l.Grammar.isAlphabetical(peek) {
		return l.lexWord()
	} else if l.Grammar.isNumeric(peek) {
		return l.lexNumber()
	} else if peek == '"' {
		return l.lexString()
	} else if l.Grammar.isOperatorRune(peek) {
		return l.lexOperator()
	} else if l.Grammar.isPunctuatorRune(peek) {
		return l.lexPunctuator()
	}

	var r rune
	var pos source.Pos

	var lexeme string
	var span source.Span

	r, pos, _, _ = l.Scanner.Next()

	// Set lexeme and set the token's span
	lexeme = string(r)
	span.Start = pos
	span.End = pos

	msg = feedback.Error{
		File: l.Scanner.File,
		What: feedback.Selection{
			Description: fmt.Sprintf("Unexpected character '%s'", lexeme),
			Span:        span,
		},
	}

	return Token{UnknownSymbol, lexeme, span}, msg
}

// Whitespace
//  - consumes any whitespace between expressions, operands, etc.
//  - inserts semicolons at the end of each line terminated by a(n):
//     - Identifier
//     - Integer or Decimal literal
//     - String literal
//     - "return" keyword
//     - "end" keyword
//     - "}" punctuator
//     - ")" punctuator
func (l *Lexer) lexWhitespace() (tok Token, msg feedback.Message) {
	var pos source.Pos
	var eol bool

	_, pos, eol, l.eof = l.Scanner.Next()

	if eol && len(l.histBuffer) > 0 {
		lastTok := l.histBuffer[len(l.histBuffer)-1]

		// Insert a semicolon into the token stream if the previous token meets
		// the right criteria
		if l.Grammar.canInsertSemicolonAfter(lastTok) {
			// Make the synthetic semicolon token and gives it an overlapping
			// position with the newline character
			return Token{
				Symbol: TokenSymbol(";"),
				Lexeme: ";",
				Span:   source.Span{Start: pos, End: pos},
			}, nil
		}
	}

	return l.readNextToken()
}

// Comments
//  - \#[^\n]*
func (l *Lexer) lexComment() (tok Token, msg feedback.Message) {
	var eol bool

	// To lex a comment, consume ALL runes after the comment's start until
	// the end of the line or file is reached (whichever is sooner)
	for {
		_, _, eol, _ = l.Scanner.Peek()

		// Escape the comment lexer before an EOL rune so that its lexing will
		// be defered to the whitespace lexing function which will handle any
		// necessary semicolon insertion
		if eol {
			break
		}

		_, _, _, l.eof = l.Scanner.Next()
	}

	return l.readNextToken()
}

// Identifiers and Keywords
//  - match [A-Za-z][A-Za-z0-9]*
func (l *Lexer) lexWord() (tok Token, msg feedback.Message) {
	var r, peek rune
	var pos source.Pos
	var eol bool

	var sym TokenSymbol
	var lexeme string
	var span source.Span

	for {
		r, pos, eol, l.eof = l.Scanner.Next()

		// If lexing just began, set the start position
		if len(lexeme) == 0 {
			span.Start = pos
		}

		// Append rune to lexeme and expand the token's span
		lexeme += string(r)
		span.End = pos

		// Exit the loop if the rune was the EOL or the EOF
		if eol || l.eof {
			break
		}

		// Peek at the upcoming rune
		peek, _, _, _ = l.Scanner.Peek()

		// Continue lexing if the upcoming rune is alphabetical
		if l.Grammar.isAlphabetical(peek) {
			continue
		}

		// Continue lexing if the upcoming rune is numeric
		if l.Grammar.isNumeric(peek) {
			continue
		}

		// Upcoming rune won't work so escape the loop
		break
	}

	// Determine whether the word classifies as a keyword or boolean literal
	// recognized by the grammar. If it does, set the appropriate token symbol
	if l.Grammar.isKeyword(lexeme) {
		sym = TokenSymbol(lexeme)
	} else if l.Grammar.isBoolean(lexeme) {
		sym = BooleanSymbol
	} else {
		sym = IdentSymbol
	}

	return Token{sym, lexeme, span}, nil
}

// Integer or Decimal literals
//  - integer match [0-9]+
//  - decimal match [0-9]+(\.[0-9]+)?
func (l *Lexer) lexNumber() (tok Token, msg feedback.Message) {
	var r, peek rune
	var pos source.Pos
	var eol bool

	var sym TokenSymbol
	var lexeme string
	var span source.Span

	sym = IntegerSymbol

	for {
		r, pos, eol, l.eof = l.Scanner.Next()

		// If lexing just began, set the start position
		if len(lexeme) == 0 {
			span.Start = pos
		}

		// Append rune to lexeme and expand the token's span
		lexeme += string(r)
		span.End = pos

		// Handle the lexing of decimal points
		if r == '.' {
			if sym == IntegerSymbol {
				// Partial token was being classified as an Integer, switch to
				// Decimal classification since a decimal point was found
				sym = DecimalSymbol
			} else {
				// Partial token was already being classified as a Decimal, so
				// finding a second decimal point is a syntax error
				msg := feedback.Error{
					Classification: feedback.SyntaxError,
					File:           l.Scanner.File,
					What: feedback.Selection{
						Description: "Second decimal point in number literal",
						// The error only highlights the second decimal point,
						// not the entire token
						Span: source.Span{Start: pos, End: pos},
					},
				}

				return Token{sym, lexeme, span}, msg
			}
		}

		// Exit the loop if the rune was the EOL or the EOF
		if eol || l.eof {
			break
		}

		// Peek at the upcoming rune
		peek, _, _, _ = l.Scanner.Peek()

		// Continue lexing if the upcoming rune is numeric
		if l.Grammar.isNumeric(peek) {
			continue
		}

		if peek == '_' {
			// Consume the upcoming underscore and DON'T append it to the lexeme
			r, pos, eol, l.eof = l.Scanner.Next()
			span.End = pos
			peek, _, _, _ = l.Scanner.Peek()

			if l.Grammar.isNumeric(peek) {
				continue
			} else {
				if peek == '_' {
					_, peekPos, _, _ := l.Scanner.Peek()

					msg = feedback.Error{
						Classification: feedback.SyntaxError,
						File:           l.Scanner.File,
						What: feedback.Selection{
							Description: "Illegal double underscore in number literal",
							Span:        source.Span{pos, peekPos},
						},
					}
				} else {
					msg = feedback.Error{
						Classification: feedback.SyntaxError,
						File:           l.Scanner.File,
						What: feedback.Selection{
							Description: "Unexpected end of number literal",
							Span:        source.Span{pos, pos},
						},
					}
				}

				return Token{sym, lexeme, span}, msg
			}
		}

		// Continue lexing if the upcoming rune is a decimal point
		if peek == '.' {
			continue
		}

		// Upcoming rune won't work so escape the loop
		break
	}

	return Token{sym, lexeme, span}, nil
}

// String literal
//  - match double quoted string, ignores escaped quotes
//  - expressions can be embedded inside of strings via the interpolation
//    syntax: `"string... \(2 + 2) string \(foo(4, 5, 6)) ... string"`
//  - the interpolation syntax is patterned off of Swift's string syntax
func (l *Lexer) lexString() (tok Token, msg feedback.Message) {
	var lexeme string
	var span source.Span

	startingRune, startingPos, _, _ := l.Scanner.Peek()
	span.Start = startingPos
	span.End = startingPos

	if l.forceStringLexing {
		l.forceStringLexing = false

		if startingRune == '"' {
			l.Scanner.Next()
			return Token{StringSymbol, "", span}, nil
		}
	}

	// String starting the normal way (as opposed to the lexing of strings
	// that conclude string interpolation expressions)
	if startingRune == '"' {
		l.Scanner.Next()
	}

	for {
		var r rune
		var pos source.Pos
		var eol bool

		r, pos, eol, l.eof = l.Scanner.Next()

		// Return with an error if the rune was the EOL or the EOF on an
		// unterminated string literal
		if eol || l.eof {
			return Token{StringSymbol, lexeme, span}, feedback.Error{
				Classification: feedback.SyntaxError,
				File:           l.Scanner.File,
				What: feedback.Selection{
					Description: "Unterminated string <middle rune>",
					Span:        span,
				},
			}
		}

		if r == '\\' {
			r, pos, eol, l.eof = l.Scanner.Next()

			if eol || l.eof {
				return Token{StringSymbol, lexeme, span}, feedback.Error{
					Classification: feedback.SyntaxError,
					File:           l.Scanner.File,
					What: feedback.Selection{
						Description: "Unterminated string",
						Span:        span,
					},
				}
			}

			switch r {
			case '\\':
				lexeme += "\\\\"
				span.End = pos
			case 'n':
				lexeme += "\\n"
				span.End = pos
			case '(':
				// Create a special token to denote the beginning of a string
				// interpolation. Defer this token with the lexer so it will be
				// emitted after the current string token is returned
				l.deferToken(Token{
					LInterpolSymbol,
					"\\(",
					source.Span{
						source.Pos{pos.Line, pos.Col-1},
						pos,
					},
				}, nil)

				// Push a new paren counter onto the paren counting stack to
				// keep track of the matched paren pairs inside the string
				// interpolation. When a `)` is reached and this counter is 0,
				// then it is time to close the interpolation expression
				l.openParenStack = append([]int{0}, l.openParenStack...)
				l.openStrings++

				// Return the current string token after the interpolation
				// infrastructure has been initialized
				return Token{StringSymbol, lexeme, span}, nil
			default:
				return Token{StringSymbol, lexeme, span}, feedback.Error{
					Classification: feedback.SyntaxError,
					File:           l.Scanner.File,
					What: feedback.Selection{
						Description: "Unknown escape sequence",
						Span:        span,
					},
				}
			}
		}

		if r == '"' {
			// Catch the closing quote and stop this lexing loop
			span.End = pos
			break
		}

		// Increment token terminal position and add the character to the lexeme
		span.End = pos
		lexeme += string(r)
	}

	return Token{StringSymbol, lexeme, span}, nil
}

// Operators
//  - consecutive operator runes are glued together until a rune is found that
//    can't be used in a valid operator
func (l *Lexer) lexOperator() (tok Token, msg feedback.Message) {
	var r, peek rune
	var pos source.Pos
	var eol bool

	var lexeme string
	var span source.Span

	for {
		r, pos, eol, l.eof = l.Scanner.Next()

		// If lexing just began, set the start position
		if len(lexeme) == 0 {
			span.Start = pos
		}

		// Append rune to lexeme and expand the token's span
		lexeme += string(r)
		span.End = pos

		// Exit the loop if the rune was the EOL or the EOF
		if eol || l.eof {
			break
		}

		peek, _, _, _ = l.Scanner.Peek()

		// Continue lexing if the upcoming rune is also a valid operator
		if l.Grammar.isOperatorRune(peek) {
			continue
		}

		break
	}

	return Token{TokenSymbol(lexeme), lexeme, span}, nil
}

// Punctuators
//  - always consist of a single character
//  - this function also works in tanden with `lexString` to determine
//    when a right-paren should be used to close a string interpolation
//    expression
func (l *Lexer) lexPunctuator() (tok Token, msg feedback.Message) {
	var r rune
	var pos source.Pos

	var lexeme string
	var span source.Span

	r, pos, _, l.eof = l.Scanner.Next()

	// This statement keeps track of paired parentheses so that during string
	// interpolation the correct right-paren is used to close the interpolation
	// expression
	if r == '(' {
		l.openParenStack[0]++
	} else if r == ')' {
		// If some string is being interpolated and all inner parentheses are
		// paried off, then it is OK to treat this left-paren as the closing
		// interpolation parenthese.
		if l.openParenStack[0] == 0 && l.openStrings > 0 {
			// Since an interpolation string is being closed, decrement the
			// counter counting the number of currently open interpolation
			// strings
			l.openStrings--

			// Set this flag to force the string lexing function to handle the
			// following characters
			l.forceStringLexing = true

			// Return the token that signals the closure of an interpolated
			// expression
			return Token{
				RInterpolSymbol,
				")",
				source.Span{pos, pos},
			}, nil
		} else {
			// Decrement paren counter if an interpolated string can't be closed
			// but since a parenthese pair was closed
			l.openParenStack[0]--
		}
	}

	// Set lexeme and set the token's span
	lexeme = string(r)
	span.Start = pos
	span.End = pos

	return Token{TokenSymbol(lexeme), lexeme, span}, nil
}

// Peek returns the next token WITHOUT advancing the lexer. Once the next token
// has been peek'ed it is cached in the Lexer so repeated calls to Peek will not
// do duplicate lexing work
func (l *Lexer) Peek() (tok Token, msg feedback.Message) {
	if len(l.peekBuffer) > 0 {
		tok = l.peekBuffer[0]
		msg = nil
	} else {
		tok, msg = l.readNextToken()
		l.peekBuffer = append(l.peekBuffer, tok)
	}

	return tok, msg
}

// PeekMatches returns true if the upcoming token matches a given TokenSymbol
func (l *Lexer) PeekMatches(sym TokenSymbol) (matches bool) {
	if tok, err := l.Peek(); err == nil {
		return tok.Symbol == sym
	}

	return false
}

// Next returns the upcoming token and advances the Lexer. If the token buffer
// contains any tokens (like those already generated by a Peek call), those
// tokens will be removed from the buffer and returned by Next
func (l *Lexer) Next() (tok Token, msg feedback.Message) {
	if len(l.peekBuffer) > 0 {
		tok = l.peekBuffer[0]
		msg = nil

		// delete token at peekBuffer[0]
		l.peekBuffer = append(l.peekBuffer[:0], l.peekBuffer[0+1:]...)
	} else {
		tok, msg = l.readNextToken()
	}

	l.histBuffer = append(l.histBuffer, tok)
	return tok, msg
}

// ExpectNext returns the next token if it matches the given TokenSymbol. If the
// upcoming token DOESN'T match, an error is also returned
func (l *Lexer) ExpectNext(sym TokenSymbol) (tok Token, msg feedback.Message) {
	if tok, msg = l.Next(); msg != nil {
		return tok, msg
	}

	if tok.Symbol == sym {
		return tok, nil
	}

	msg = feedback.Error{
		File: l.Scanner.File,
		What: feedback.Selection{
			Description: fmt.Sprintf(
				"Expected '%s' instead found '%s'",
				sym,
				tok.Symbol),
			Span: tok.Span,
		},
	}

	return tok, msg
}

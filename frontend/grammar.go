package frontend

// Grammar holds a collection of helper methods for classifying runes and
// keywords based on a given language specification
type Grammar struct {
	OperatorRunes   []rune
	PunctuatorRunes []rune
	Keywords        []string
}

func (g *Grammar) isCommentStart(r rune) (matches bool) {
	return (r == '#')
}

func (g *Grammar) isValidLineBreak(r rune) (matches bool) {
	return (r == '\n')
}

func (g *Grammar) isWhitespace(r rune) (matches bool) {
	return (r <= ' ')
}

func (g *Grammar) isAlphabetical(r rune) (matches bool) {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func (g *Grammar) isNumeric(r rune) (matches bool) {
	return (r >= '0' && r <= '9')
}

// isOperatorRune returns true if a given rune is included in the Grammar's list
// of valid operator runes
func (g *Grammar) isOperatorRune(r rune) (matches bool) {
	for i, l := 0, len(g.OperatorRunes); i < l; i++ {
		if g.OperatorRunes[i] == r {
			return true
		}
	}

	return false
}

// isPunctuatorRune returns true if a given rune is included in the Grammar's list
// of valid punctuation runes
func (g *Grammar) isPunctuatorRune(r rune) (matches bool) {
	for i, l := 0, len(g.PunctuatorRunes); i < l; i++ {
		if g.PunctuatorRunes[i] == r {
			return true
		}
	}

	return false
}

// isKeyword returns true if a given string is included in the Grammar's list
// of valid keywords
func (g *Grammar) isKeyword(s string) (matches bool) {
	for i, l := 0, len(g.Keywords); i < l; i++ {
		if g.Keywords[i] == s {
			return true
		}
	}

	return false
}

// canInsertSemicolonAfter returns true if a given token can be the terminal token
// in a statement or expression this includes:
//   - Identifier
//   - Integer or Decimal literal
//   - String literal
//   - "return" keyword
//   - "end" keyword
//   - "}" punctuator
//   - ")" punctuator
func (g *Grammar) canInsertSemicolonAfter(tok Token) (matches bool) {
	return (tok.Symbol == IdentSymbol ||
		tok.Symbol == IntegerSymbol ||
		tok.Symbol == DecimalSymbol ||
		tok.Symbol == StringSymbol ||
		(tok.Symbol == TokenSymbol("return")) ||
		(tok.Symbol == TokenSymbol("end")) ||
		(tok.Symbol == TokenSymbol("}")) ||
		(tok.Symbol == TokenSymbol(")")))
}

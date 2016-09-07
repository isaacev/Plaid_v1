package frontend

import (
	"github.com/isaacev/Plaid/source"
)

// TokenSymbol is the classification system for tokens. Identifier and literal
// tokens are represented by general token symbols (like "Ident") while operator
// and punctuation tokens are represented by their literal values
type TokenSymbol string

// Token structs represent a lexical atom and are tagged with a token symbol
// classification, and source code line/column data
type Token struct {
	Symbol TokenSymbol
	Lexeme string
	Span   source.Span
}

// The most common token symbols are defined as part of the "frontend" package
const (
	EOFSymbol      TokenSymbol = "EOF"
	UnknownSymbol  TokenSymbol = "Unknown Token"
	LBracketSymbol TokenSymbol = "["
	RBracketSymbol TokenSymbol = "]"
	LParenSymbol   TokenSymbol = "("
	RParenSymbol   TokenSymbol = ")"
	CommaSymbol    TokenSymbol = ","
	ColonSymbol    TokenSymbol = ":"
	IdentSymbol    TokenSymbol = "Identifier"
	IntegerSymbol  TokenSymbol = "Integer"
	BooleanSymbol  TokenSymbol = "Boolean"
	DecimalSymbol  TokenSymbol = "Decimal"
	StringSymbol   TokenSymbol = "String"
)

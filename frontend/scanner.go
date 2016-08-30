package frontend

import (
	"unicode/utf8"

	"github.com/isaacev/Plaid/source"
)

/**
 * # Handling of Line & File terminations
 *
 * The first character in each line is considered to be in column 1. A newline
 * at the end of a line with `N` characters is considered to be in column
 * `N + 1`.
 *
 * The scanner's handling of end-of-file conditions is a little more complex.
 * Since the EOF is not denoted by a specific rune as is possible with carraige-
 * return & line-feed, the EOL and EOF flags are set for the last rune in the
 * document no matter what the rune is. Any subsequent calls to `Next()` or
 * `Peek()` will panic.
 */

// Scanner strcts hold the state of a scanner instance which consumes source
// code runes one at a time. Since source code documents can be Unicode, the
// scanner must keep track of each rune's byte offset. The scanner also records
// line and column data which it emits along with each rune.
type Scanner struct {
	File     *source.File
	nextByte int // initialized to 0
	nextLine int // ...  ...  ...  1
	nextCol  int // ...  ...  ...  0
}

// NewScanner is a basic constructor function for Scanners which populates
// private fields with the appropriate starting values
func NewScanner(file *source.File) *Scanner {
	return &Scanner{
		File:     file,
		nextByte: 0,
		nextLine: 1,
		nextCol:  1,
	}
}

// Peek returns the next rune, next rune position and an end-of-file flag. To
// the user the Scanner will not seem to have advanced even though it does
// internally. This is because Peek reads are cahced so that the Scanner
// internals need not be decremented and so that subsequent Peek & Next calls
// can use cached data instead of reading a rune again
func (s *Scanner) Peek() (r rune, pos source.Pos, EOL bool, EOF bool) {
	if s.nextByte >= len(s.File.Contents) {
		panic("attempt to scan past EOF")
	}

	// Extract the next rune from the document buffer
	runeValue, runeWidth := utf8.DecodeRuneInString(s.File.Contents[s.nextByte:])

	// Use `nextLine`, `nextCol` values as rune position values
	pos.Line = s.nextLine
	pos.Col = s.nextCol

	// Set EOL flag
	EOL = (runeValue == '\n')

	// Set EOF flag
	EOF = (s.nextByte+runeWidth == len(s.File.Contents))

	return runeValue, pos, EOL, EOF
}

// Next returns the next rune, the rune's position and an end-of-file flag. If
// cached data exists, that is returne and the cache is emptied. A call to Next
// will advance the Scanner permanently.
func (s *Scanner) Next() (r rune, pos source.Pos, EOL bool, EOF bool) {
	if s.nextByte >= len(s.File.Contents) {
		panic("attempt to scan past EOF")
	}

	// Extract the next rune from the document buffer
	runeValue, runeWidth := utf8.DecodeRuneInString(s.File.Contents[s.nextByte:])

	// Use `nextLine`, `nextCol` values as rune position values
	pos.Line = s.nextLine
	pos.Col = s.nextCol

	// Update `nextLine`, `nextCol`
	if runeValue == '\n' {
		s.nextLine++
		s.nextCol = 1
	} else {
		s.nextCol++
	}

	// Set EOL flag
	EOL = (runeValue == '\n')

	// Set EOF flag
	EOF = (s.nextByte+runeWidth == len(s.File.Contents))

	// Update `nextByte` to account for byte width of this rune
	s.nextByte += runeWidth

	return runeValue, pos, EOL, EOF
}

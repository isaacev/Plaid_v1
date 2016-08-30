package source

// Pos holds the line/column data for a single rune in a source code document
type Pos struct {
	Line int
	Col  int
}

// Span holds a Start and End position in a source code document
type Span struct {
	Start Pos
	End   Pos
}

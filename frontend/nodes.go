package frontend

import (
	"github.com/isaacev/Plaid/source"
)

// Node is a generic node in the abstract syntax tree (AST)
type Node interface {
	Pos() source.Pos
	End() source.Pos
}

// Expr represents a Node that returns a value when executed
type Expr interface {
	Node
	Type() *Type
	exprNode()
}

// Stmt represents a Node that does not necessarily return a value when executed
type Stmt interface {
	Node
	stmtNode()
}

// Program is the root node for an AST
type Program struct {
	Statements []Stmt

	// this field is populated during the type-checking stage where scope
	// analysis is performed and local variables can be easily counted
	Locals   []*LocalRecord
	Upvalues []*UpvalueRecord
}

// Pos returns the starting source code position of this node
func (p Program) Pos() source.Pos {
	if len(p.Statements) > 0 {
		return p.Statements[0].Pos()
	}

	return source.Pos{
		Line: 1,
		Col:  1,
	}
}

// End returns the terminal source code position of this node
func (p Program) End() source.Pos {
	if len(p.Statements) > 0 {
		return p.Statements[len(p.Statements)-1].End()
	}

	return source.Pos{
		Line: 1,
		Col:  1,
	}
}

// FuncExpr represents an anonymous function definition
type FuncExpr struct {
	Parameters       *FieldList
	ReturnAnnotation *IdentExpr
	Body             *FunctionBody

	// this field is populated during the type-checking stage where scope
	// analysis is performed and local variables can be easily counted
	Locals   []*LocalRecord
	Upvalues []*UpvalueRecord
	t        *Type
}

func (i FuncExpr) Type() *Type {
	return i.t
}

// Pos returns the starting source code position of this node
func (f FuncExpr) Pos() source.Pos {
	return f.Parameters.Pos()
}

// End returns the terminal source code position of this node
func (f FuncExpr) End() source.Pos {
	return f.Body.End()
}

func (f FuncExpr) exprNode() {}
func (f FuncExpr) stmtNode() {}

// FunctionBody represents the collection of statements that make up part of a
// function definition
type FunctionBody struct {
	Statements []Stmt
	LeftBrace  Token
	RightBrace Token
}

// Pos returns the starting source code position of this node
func (f FunctionBody) Pos() source.Pos {
	return f.LeftBrace.Span.Start
}

// End returns the terminal source code position of this node
func (f FunctionBody) End() source.Pos {
	return f.RightBrace.Span.Start
}

// IfStmt represents a basic conditional statement
type IfStmt struct {
	IfKeyword Token
	Condition Expr
	Body      *ConditionalBody
}

// Pos returns the starting source code position of this node
func (i IfStmt) Pos() source.Pos {
	return i.IfKeyword.Span.Start
}

// End returns the terminal source code position of this node
func (i IfStmt) End() source.Pos {
	return i.Body.End()
}

func (i IfStmt) stmtNode() {}

// LoopStmt represents a loop statement
type LoopStmt struct {
	LoopKeyword Token
	Condition   Expr
	Body        *ConditionalBody
}

// Pos returns the starting source code position of this node
func (w LoopStmt) Pos() source.Pos {
	return w.LoopKeyword.Span.Start
}

// End returns the terminal source code position of this node
func (w LoopStmt) End() source.Pos {
	return w.Body.End()
}

func (w LoopStmt) stmtNode() {}

// ConditionalBody represents the collection of statements that make up the
// bodies of conditional statements like If and Loop
type ConditionalBody struct {
	Statements []Stmt
	Colon      Token
	EndKeyword Token
}

// Pos returns the starting source code position of this node
func (c ConditionalBody) Pos() source.Pos {
	return c.Colon.Span.Start
}

// End returns the terminal source code position of this node
func (c ConditionalBody) End() source.Pos {
	return c.EndKeyword.Span.Start
}

// FieldList represents a collection of type annotations like those found in
// function parameter definitions
type FieldList struct {
	Fields     []*TypeAnnotationStmt
	LeftParen  Token
	RightParen Token
}

// Pos returns the starting source code position of this node
func (f FieldList) Pos() source.Pos {
	return f.LeftParen.Span.Start
}

// End returns the terminal source code position of this node
func (f FieldList) End() source.Pos {
	return f.RightParen.Span.Start
}

// PrintStmt represents a print statement which outputs the result of an
// expression followed by a newline
type PrintStmt struct {
	PrintKeyword Token
	Arguments    []Expr
}

// Pos returns the starting source code position of this node
func (p PrintStmt) Pos() source.Pos {
	return p.PrintKeyword.Span.Start
}

// End returns the terminal source code position of this node
func (p PrintStmt) End() source.Pos {
	if len(p.Arguments) == 0 {
		return p.PrintKeyword.Span.End
	}

	return p.Arguments[len(p.Arguments)-1].End()
}

func (p PrintStmt) stmtNode() {}

// DispatchExpr represents a function dispatch including root and any arguments
type DispatchExpr struct {
	Root       *IdentExpr
	Arguments  []Expr
	LeftParen  Token
	RightParen Token
	t          *Type
}

func (i DispatchExpr) Type() *Type {
	return i.t
}

// Pos returns the starting source code position of this node
func (d DispatchExpr) Pos() source.Pos {
	return d.Root.Pos()
}

// End returns the terminal source code position of this node
func (d DispatchExpr) End() source.Pos {
	return d.RightParen.Span.End
}

func (d DispatchExpr) exprNode() {}
func (d DispatchExpr) stmtNode() {}

// ReturnStmt represents a print statement which outputs the result of an
// expression followed by a newline
type ReturnStmt struct {
	ReturnKeyword Token
	Argument      Expr
}

// Pos returns the starting source code position of this node
func (p ReturnStmt) Pos() source.Pos {
	return p.ReturnKeyword.Span.Start
}

// End returns the terminal source code position of this node
func (p ReturnStmt) End() source.Pos {
	if p.Argument == nil {
		return p.ReturnKeyword.Span.End
	}

	return p.Argument.End()
}

func (p ReturnStmt) stmtNode() {}

// BinaryExpr represents a basic expression of the form:
// <left expr> <operator> <right expr>
type BinaryExpr struct {
	Operator TokenSymbol
	Left     Expr
	Right    Expr
	t        *Type
}

func (b BinaryExpr) Type() *Type {
	return b.t
}

// Pos returns the starting source code position of this node
func (b BinaryExpr) Pos() source.Pos {
	return b.Left.Pos()
}

// End returns the terminal source code position of this node
func (b BinaryExpr) End() source.Pos {
	return b.Right.End()
}

func (b BinaryExpr) exprNode() {}
func (b BinaryExpr) stmtNode() {}

// TypeAnnotationStmt represents the association of a variable name with a type
type TypeAnnotationStmt struct {
	Identifier   *IdentExpr
	Annotation   *IdentExpr
	ExplicitType bool
}

// Pos returns the starting source code position of this node
func (t TypeAnnotationStmt) Pos() source.Pos {
	return t.Identifier.Pos()
}

// End returns the terminal source code position of this node
func (t TypeAnnotationStmt) End() source.Pos {
	if t.Annotation == nil {
		return t.Identifier.End()
	}

	return t.Annotation.End()
}

func (t TypeAnnotationStmt) stmtNode() {}

// DeclarationStmt represents the mapping of a value to a variable
type DeclarationStmt struct {
	LetKeyword Token
	Assignee   *IdentExpr
	Assignment Expr
}

// Pos returns the starting source code position of this node
func (a DeclarationStmt) Pos() source.Pos {
	return a.LetKeyword.Span.Start
}

// End returns the terminal source code position of this node
func (a DeclarationStmt) End() source.Pos {
	return a.Assignment.End()
}

func (a DeclarationStmt) stmtNode() {}

// AssignmentStmt represents the mapping of a value to a variable
type AssignmentStmt struct {
	Assignee   *IdentExpr
	Assignment Expr
}

// Pos returns the starting source code position of this node
func (a AssignmentStmt) Pos() source.Pos {
	return a.Assignee.Pos()
}

// End returns the terminal source code position of this node
func (a AssignmentStmt) End() source.Pos {
	return a.Assignment.End()
}

func (a AssignmentStmt) stmtNode() {}

// IdentExpr represents a single identifier in the AST
type IdentExpr struct {
	NamePos source.Pos
	Name    string
	t       *Type
}

func (i IdentExpr) Type() *Type {
	return i.t
}

// Pos returns the starting source code position of this node
func (i IdentExpr) Pos() source.Pos {
	return i.NamePos
}

// End returns the terminal source code position of this node
func (i IdentExpr) End() source.Pos {
	return source.Pos{
		Line: i.NamePos.Line,
		Col:  i.NamePos.Col + len(i.Name) - 1,
	}
}

func (i IdentExpr) exprNode() {}
func (i IdentExpr) stmtNode() {}

// Literal node represents a literal value in the AST
type Literal interface {
	Expr
	literalNode()
}

// IntegerExpr represents an instance of an integer literal in the AST
type IntegerExpr struct {
	Lexeme  string
	Value   int32
	Start   source.Pos
	t       *Type
}

func (i IntegerExpr) Type() *Type {
	return i.t
}

// Pos returns the starting source code position of this node
func (i IntegerExpr) Pos() source.Pos {
	return i.Start
}

// End returns the terminal source code position of this node
func (i IntegerExpr) End() source.Pos {
	return source.Pos{
		Line: i.Start.Line,
		Col:  i.Start.Col + (len(i.Lexeme) - 1),
	}
}

func (i IntegerExpr) literalNode() {}
func (i IntegerExpr) exprNode()    {}
func (i IntegerExpr) stmtNode()    {}

// DecimalExpr represents an instance of a floating point literal in the AST
type DecimalExpr struct {
	Lexeme  string
	Value   float32
	Start   source.Pos
	t       *Type
}

func (i DecimalExpr) Type() *Type {
	return i.t
}

// Pos returns the starting source code position of this node
func (i DecimalExpr) Pos() source.Pos {
	return i.Start
}

// End returns the terminal source code position of this node
func (i DecimalExpr) End() source.Pos {
	return source.Pos{
		Line: i.Start.Line,
		Col:  i.Start.Col + (len(i.Lexeme) - 1),
	}
}

func (i DecimalExpr) literalNode() {}
func (i DecimalExpr) exprNode()    {}
func (i DecimalExpr) stmtNode()    {}

// StringExpr represents an instance of a string literal in the AST
type StringExpr struct {
	Lexeme  string
	Value   string
	Start   source.Pos
	t       *Type
	casting *Type
}

func (i StringExpr) IsCast() (bool, *Type) {
	if i.casting != nil {
		return true, i.casting
	}

	return false, nil
}

func (i StringExpr) Type() *Type {
	return i.t
}

// Pos returns the starting source code position of this node
func (s StringExpr) Pos() source.Pos {
	return s.Start
}

// End returns the terminal source code position of this node
func (s StringExpr) End() source.Pos {
	return source.Pos{
		Line: s.Start.Line,
		Col:  s.Start.Col + (len(s.Lexeme) - 1),
	}
}

func (s StringExpr) literalNode() {}
func (s StringExpr) exprNode()    {}
func (s StringExpr) stmtNode()    {}

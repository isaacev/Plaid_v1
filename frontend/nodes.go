package frontend

import (
	"github.com/isaacev/Plaid/source"
)

// Node is a generic node in the abstract syntax tree (AST)
type Node interface {
	Pos() source.Pos
	End() source.Pos
}

/**
 * ROOT PROGRAM NODE
 */

// ProgramNode is the root node for an AST
type ProgramNode struct {
	Statements []Stmt

	// this field is populated during the type-checking stage where scope
	// analysis is performed and local variables can be easily counted
	Locals   []*LocalRecord
	Upvalues []*UpvalueRecord
}

// Pos returns the starting source code position of this node
func (p ProgramNode) Pos() source.Pos {
	if len(p.Statements) > 0 {
		return p.Statements[0].Pos()
	}

	return source.Pos{1, 1}
}

// End returns the terminal source code position of this node
func (p ProgramNode) End() source.Pos {
	if len(p.Statements) > 0 {
		return p.Statements[len(p.Statements)-1].End()
	}

	return source.Pos{1, 1}
}

/**
 * STATEMENT NODES
 *  - statements--unlike expressions--emit no useful value and thus cannot be
 *    used inside of expressions
 *  - expressions can be used as statements but their emitted value is discarded
 */

// Stmt represents a Node that does not necessarily return a value when executed
type Stmt interface {
	Node
	stmtNode()
}

// IfStmt represents a basic conditional statement
type IfStmt struct {
	IfClause    *Clause
	ElifClauses []*Clause
	ElseClause  *Clause
	EndKeyword  Token
}

// Pos returns the starting source code position of this node
func (i IfStmt) Pos() source.Pos {
	return i.IfClause.Pos()
}

// End returns the terminal source code position of this node
func (i IfStmt) End() source.Pos {
	return i.EndKeyword.Span.End
}

func (IfStmt) stmtNode() {}

// LoopStmt represents a loop statement
type LoopStmt struct {
	Clause     *Clause
	EndKeyword Token
}

// Pos returns the starting source code position of this node
func (l LoopStmt) Pos() source.Pos {
	return l.Clause.Pos()
}

// End returns the terminal source code position of this node
func (l LoopStmt) End() source.Pos {
	return l.EndKeyword.Span.End
}

func (LoopStmt) stmtNode() {}

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

func (DeclarationStmt) stmtNode() {}

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

func (AssignmentStmt) stmtNode() {}

// ReturnStmt represents a print statement which outputs the result of an
// expression followed by a newline
type ReturnStmt struct {
	ReturnKeyword Token
	Argument      Expr
}

// Pos returns the starting source code position of this node
func (r ReturnStmt) Pos() source.Pos {
	return r.ReturnKeyword.Span.Start
}

// End returns the terminal source code position of this node
func (r ReturnStmt) End() source.Pos {
	if r.Argument == nil {
		return r.ReturnKeyword.Span.End
	}

	return r.Argument.End()
}

func (ReturnStmt) stmtNode() {}

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

func (PrintStmt) stmtNode() {}

/**
 * EXPRESSION NODES
 *  - expressions emit some value with a type so they can be composed
 *    recursively to make more complex expressions or statements
 *  - each expression has a `_type` field which is populated during the type
 *    checking phase with the computed type output by the expression based on
 *    the rules in the scope's type-table
 */

// Expr represents a Node that returns a value when executed
type Expr interface {
	Node
	GetType() Type
	SetType(Type)
	exprNode()
}

// DispatchExpr represents a function dispatch including root and any arguments
type DispatchExpr struct {
	Root       Expr
	Arguments  []Expr
	LeftParen  Token
	RightParen Token
	_type      Type
}

// SetType populates the `_type` field of this expression (this is done during
// the type-checking phase)
func (d *DispatchExpr) SetType(_type Type) {
	d._type = _type
}

// GetType returns the Type associated with this expression. This should never
// be called before the expression has been type checked since it will return
// `nil` in that case
func (d *DispatchExpr) GetType() Type {
	return d._type
}

// Pos returns the starting source code position of this node
func (d *DispatchExpr) Pos() source.Pos {
	return d.Root.Pos()
}

// End returns the terminal source code position of this node
func (d *DispatchExpr) End() source.Pos {
	return d.RightParen.Span.End
}

func (*DispatchExpr) exprNode() {}
func (*DispatchExpr) stmtNode() {}

// IndexAccessExpr represents an expression to extract an element from some list
// at a particular index
type IndexAccessExpr struct {
	Root         Expr
	LeftBracket  Token
	Index        Expr
	RightBracket Token
	_type        Type
}

// SetType populates the `_type` field of this expression (this is done during
// the type-checking phase)
func (i *IndexAccessExpr) SetType(_type Type) {
	i._type = _type
}

// GetType returns the Type associated with this expression. This should never
// be called before the expression has been type checked since it will return
// `nil` in that case
func (i *IndexAccessExpr) GetType() Type {
	return i._type
}

// Pos returns the starting source code position of this node
func (i *IndexAccessExpr) Pos() source.Pos {
	return i.Root.Pos()
}

// End returns the terminal source code position of this node
func (i *IndexAccessExpr) End() source.Pos {
	return i.RightBracket.Span.End
}

func (*IndexAccessExpr) exprNode() {}
func (*IndexAccessExpr) stmtNode() {}

// UnaryExpr represents a basic expression of the form:
// <operator> <operand>
type UnaryExpr struct {
	Operator Token
	Operand  Expr
	_type    Type
}

// SetType populates the `_type` field of this expression (this is done during
// the type-checking phase)
func (u *UnaryExpr) SetType(_type Type) {
	u._type = _type
}

// GetType returns the Type associated with this expression. This should never
// be called before the expression has been type checked since it will return
// `nil` in that case
func (u *UnaryExpr) GetType() Type {
	return u._type
}

// Pos returns the starting source code position of this node
func (u *UnaryExpr) Pos() source.Pos {
	return u.Operator.Span.Start
}

// End returns the terminal source code position of this node
func (u *UnaryExpr) End() source.Pos {
	return u.Operand.End()
}

func (*UnaryExpr) exprNode() {}
func (*UnaryExpr) stmtNode() {}

// BinaryExpr represents a basic expression of the form:
// <left expr> <operator> <right expr>
type BinaryExpr struct {
	Operator TokenSymbol
	Left     Expr
	Right    Expr
	_type    Type
}

// SetType populates the `_type` field of this expression (this is done during
// the type-checking phase)
func (b *BinaryExpr) SetType(_type Type) {
	b._type = _type
}

// GetType returns the Type associated with this expression. This should never
// be called before the expression has been type checked since it will return
// `nil` in that case
func (b *BinaryExpr) GetType() Type {
	return b._type
}

// Pos returns the starting source code position of this node
func (b *BinaryExpr) Pos() source.Pos {
	return b.Left.Pos()
}

// End returns the terminal source code position of this node
func (b *BinaryExpr) End() source.Pos {
	return b.Right.End()
}

func (*BinaryExpr) exprNode() {}
func (*BinaryExpr) stmtNode() {}

// IdentExpr represents a single identifier in the AST
type IdentExpr struct {
	NamePos source.Pos
	Name    string
	_type   Type
}

// SetType populates the `_type` field of this expression (this is done during
// the type-checking phase)
func (i *IdentExpr) SetType(_type Type) {
	i._type = _type
}

// GetType returns the Type associated with this expression. This should never
// be called before the expression has been type checked since it will return
// `nil` in that case
func (i *IdentExpr) GetType() Type {
	return i._type
}

// Pos returns the starting source code position of this node
func (i *IdentExpr) Pos() source.Pos {
	return i.NamePos
}

// End returns the terminal source code position of this node
func (i *IdentExpr) End() source.Pos {
	return source.Pos{i.NamePos.Line, i.NamePos.Col + len(i.Name) - 1}
}

func (*IdentExpr) exprNode() {}
func (*IdentExpr) stmtNode() {}

/**
 * EXPRESSION LITERAL NODES
 *  - these nodes represent fundamental expressions of built in data-structures
 *    in the language. This includes number literals (both integers and
 *    decimals), strings literals, boolean keywords, and function literals
 */

// Literal node represents a literal value in the AST
type Literal interface {
	Expr
	literalNode()
}

// ListLiteral represents a collection of 0 or more values
type ListLiteral struct {
	LeftBracket  Token
	Elements     []Expr
	RightBracket Token
	_type        *ListType
}

// SetType populates the `_type` field of this expression (this is done during
// the type-checking phase)
func (l *ListLiteral) SetType(_type Type) {
	l._type = _type.(*ListType)
}

// GetType returns the Type associated with this expression. This should never
// be called before the expression has been type checked since it will return
// `nil` in that case
func (l *ListLiteral) GetType() Type {
	return l._type
}

// Pos returns the starting source code position of this node
func (l *ListLiteral) Pos() source.Pos {
	return l.LeftBracket.Span.Start
}

// End returns the terminal source code position of this node
func (l *ListLiteral) End() source.Pos {
	return l.RightBracket.Span.End
}

func (*ListLiteral) literalNode() {}
func (*ListLiteral) exprNode()    {}
func (*ListLiteral) stmtNode()    {}

// FuncLiteral represents an anonymous function definition
type FuncLiteral struct {
	FnKeyword        Token
	LeftParen        Token
	Parameters       []*Parameter
	RightParen       Token
	ReturnAnnotation TypeAnnotation
	Body             *FuncBody

	// this field is populated during the type-checking stage where scope
	// analysis is performed and local variables can be easily counted
	Locals   []*LocalRecord
	Upvalues []*UpvalueRecord
	_type    *FuncType
}

// SetType populates the `_type` field of this expression (this is done during
// the type-checking phase)
func (f *FuncLiteral) SetType(_type Type) {
	f._type = _type.(*FuncType)
}

// GetType returns the Type associated with this expression. This should never
// be called before the expression has been type checked since it will return
// `nil` in that case
func (f *FuncLiteral) GetType() Type {
	return f._type
}

// Pos returns the starting source code position of this node
func (f *FuncLiteral) Pos() source.Pos {
	return f.FnKeyword.Span.Start
}

// End returns the terminal source code position of this node
func (f *FuncLiteral) End() source.Pos {
	return f.Body.End()
}

func (*FuncLiteral) literalNode() {}
func (*FuncLiteral) exprNode()    {}
func (*FuncLiteral) stmtNode()    {}

// StrLiteral represents an instance of a string literal in the AST
type StrLiteral struct {
	Value string
	Token Token
	_type *TypeOperator
}

// SetType populates the `_type` field of this expression (this is done during
// the type-checking phase)
func (s *StrLiteral) SetType(_type Type) {
	s._type = _type.(*TypeOperator)
}

// GetType returns the Type associated with this expression. This should never
// be called before the expression has been type checked since it will return
// `nil` in that case
func (s StrLiteral) GetType() Type {
	return s._type
}

// Pos returns the starting source code position of this node
func (s *StrLiteral) Pos() source.Pos {
	return s.Token.Span.Start
}

// End returns the terminal source code position of this node
func (s *StrLiteral) End() source.Pos {
	return s.Token.Span.End
}

func (*StrLiteral) literalNode() {}
func (*StrLiteral) exprNode()    {}
func (*StrLiteral) stmtNode()    {}

// DecLiteral represents an instance of a floating point literal in the AST
type DecLiteral struct {
	Value float32
	Token Token
	_type *TypeOperator
}

// SetType populates the `_type` field of this expression (this is done during
// the type-checking phase)
func (d *DecLiteral) SetType(_type Type) {
	d._type = _type.(*TypeOperator)
}

// GetType returns the Type associated with this expression. This should never
// be called before the expression has been type checked since it will return
// `nil` in that case
func (d *DecLiteral) GetType() Type {
	return d._type
}

// Pos returns the starting source code position of this node
func (d *DecLiteral) Pos() source.Pos {
	return d.Token.Span.Start
}

// End returns the terminal source code position of this node
func (d *DecLiteral) End() source.Pos {
	return d.Token.Span.End
}

func (*DecLiteral) literalNode() {}
func (*DecLiteral) exprNode()    {}
func (*DecLiteral) stmtNode()    {}

// IntLiteral represents an instance of an integer literal in the AST
type IntLiteral struct {
	Value int32
	Token Token
	_type *TypeOperator
}

// SetType populates the `_type` field of this expression (this is done during
// the type-checking phase)
func (i *IntLiteral) SetType(_type Type) {
	i._type = _type.(*TypeOperator)
}

// GetType returns the Type associated with this expression. This should never
// be called before the expression has been type checked since it will return
// `nil` in that case
func (i *IntLiteral) GetType() Type {
	return i._type
}

// Pos returns the starting source code position of this node
func (i *IntLiteral) Pos() source.Pos {
	return i.Token.Span.Start
}

// End returns the terminal source code position of this node
func (i *IntLiteral) End() source.Pos {
	return i.Token.Span.End
}

func (*IntLiteral) literalNode() {}
func (*IntLiteral) exprNode()    {}
func (*IntLiteral) stmtNode()    {}

// BoolLiteral represents an instance of a boolean keyword literal in the AST
type BoolLiteral struct {
	Value bool
	Token Token
	_type *TypeOperator
}

// SetType populates the `_type` field of this expression (this is done during
// the type-checking phase)
func (b *BoolLiteral) SetType(_type Type) {
	b._type = _type.(*TypeOperator)
}

// GetType returns the Type associated with this expression. This should never
// be called before the expression has been type checked since it will return
// `nil` in that case
func (b *BoolLiteral) GetType() Type {
	return b._type
}

// Pos returns the starting source code position of this node
func (b *BoolLiteral) Pos() source.Pos {
	return b.Token.Span.Start
}

// End returns the terminal source code position of this node
func (b *BoolLiteral) End() source.Pos {
	return b.Token.Span.End
}

func (*BoolLiteral) literalNode() {}
func (*BoolLiteral) exprNode()    {}
func (*BoolLiteral) stmtNode()    {}

/**
 * TYPE ANNOTATION NODES
 *  - these nodes represent the structure of the type annotations used in
 *    function signatures and elsewhere
 *  - during the type-checking phase, these nodes are converted to real types
 */

// TypeAnnotation node represents any type annotation in the AST
type TypeAnnotation interface {
	Node
}

// NamedTypeAnnotation represents a type annotation that consists of only an
// identifier which will correspond to a type scope
type NamedTypeAnnotation struct {
	Name *IdentExpr
}

// Pos returns the starting source code position of this node
func (nt NamedTypeAnnotation) Pos() source.Pos {
	return nt.Name.Pos()
}

// End returns the terminal source code position of this node
func (nt NamedTypeAnnotation) End() source.Pos {
	return nt.Name.End()
}

type FuncTypeAnnotation struct {
	LeftParen  Token
	Parameters []TypeAnnotation
	RightParen Token
	ReturnType TypeAnnotation
}

// Pos returns the starting source code position of this node
func (fn FuncTypeAnnotation) Pos() source.Pos {
	// FIXME: parameters don't have to be wrapped in parantheses so the
	// LeftParen and RightParen fields might not be useful
	return fn.LeftParen.Span.Start
}

// End returns the terminal source code position of this node
func (fn FuncTypeAnnotation) End() source.Pos {
	return fn.ReturnType.End()
}

// ListTypeAnnotation represents a type annotation that corresponds the list
// where each element can be cast to `ElementType`
type ListTypeAnnotation struct {
	LeftBracket  Token
	ElementType  TypeAnnotation
	RightBracket Token
}

// Pos returns the starting source code position of this node
func (lt ListTypeAnnotation) Pos() source.Pos {
	return lt.LeftBracket.Span.Start
}

// End returns the terminal source code position of this node
func (lt ListTypeAnnotation) End() source.Pos {
	return lt.RightBracket.Span.End
}

/**
 * UTILITY NODES
 *  - utility nodes are not complete syntactic statements or expressions but are
 *    used by larger AST nodes for internal organization
 */

// Clause represents a test/body pair in flow control statements. The
// `Keyword` field is populated by tokens like `if`, `elif`, `else`, and `while`
type Clause struct {
	Keyword   Token
	Condition Expr
	Body      *ClauseBody
}

// Pos returns the starting source code position of this node
func (c Clause) Pos() source.Pos {
	return c.Keyword.Span.Start
}

// End returns the terminal source code position of this node
func (c Clause) End() source.Pos {
	return c.Body.End()
}

// ClauseBody represents the collection of statements that make up the
// bodies of conditional statements like If and Loop
type ClauseBody struct {
	Colon      Token
	Statements []Stmt
}

// Pos returns the starting source code position of this node
func (c ClauseBody) Pos() source.Pos {
	return c.Colon.Span.Start
}

// End returns the terminal source code position of this node
func (c ClauseBody) End() source.Pos {
	if len(c.Statements) > 0 {
		return c.Statements[len(c.Statements)-1].End()
	}

	return c.Colon.Span.End
}

// FuncBody represents the collection of statements that make up part of a
// function definition
type FuncBody struct {
	LeftBrace  Token
	Statements []Stmt
	RightBrace Token
}

// Pos returns the starting source code position of this node
func (f FuncBody) Pos() source.Pos {
	return f.LeftBrace.Span.Start
}

// End returns the terminal source code position of this node
func (f FuncBody) End() source.Pos {
	return f.RightBrace.Span.Start
}

// Parameter represents an Identifier and an optional TypeAnnotation with that
// identifier
type Parameter struct {
	Name       *IdentExpr
	Annotation TypeAnnotation
}

// Pos returns the starting source code position of this node
func (p Parameter) Pos() source.Pos {
	return p.Name.Pos()
}

// End returns the terminal source code position of this node
func (p Parameter) End() source.Pos {
	if p.Annotation == nil {
		return p.Name.End()
	}

	return p.Annotation.End()
}

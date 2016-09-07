package frontend

import (
	"fmt"
	"strconv"

	"github.com/isaacev/Plaid/feedback"
	"github.com/isaacev/Plaid/source"
)

func identParselet(p *Parser, tok Token) (expr Node, msg feedback.Message) {
	return &IdentExpr{
		Name:    tok.Lexeme,
		NamePos: tok.Span.Start,
	}, nil
}

func literalParselet(p *Parser, tok Token) (expr Node, msg feedback.Message) {
	const precisionBits int = 32

	switch tok.Symbol {
	case IntegerSymbol:
		// FIXME: parse more than just integers and handle this error
		tmpInt64, _ := strconv.ParseInt(tok.Lexeme, 10, precisionBits)
		i32 := int32(tmpInt64)

		return &IntLiteral{
			Lexeme: tok.Lexeme,
			Value:  i32,
			Start:  tok.Span.Start,
		}, nil
	case DecimalSymbol:
		// FIXME: parse more than just floats and handle this error
		f64, _ := strconv.ParseFloat(tok.Lexeme, precisionBits)
		f32 := float32(f64)

		return &DecLiteral{
			Lexeme: tok.Lexeme,
			Value:  f32,
			Start:  tok.Span.Start,
		}, nil
	case StringSymbol:
		var trimmed string

		// trim trailing double quote
		if last := len(tok.Lexeme) - 1; last >= 0 && tok.Lexeme[last] == '"' {
			trimmed = tok.Lexeme[:last]
		}

		// trim leading double quote
		trimmed = trimmed[1:]

		return &StrLiteral{
			Lexeme: tok.Lexeme,
			Value:  trimmed,
			Start:  tok.Span.Start,
		}, nil
	default:
		return nil, feedback.Error{
			Classification: feedback.SyntaxError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("Unexpected symbol `%s`", tok.Symbol),
				Span:        tok.Span,
			},
		}
	}
}

func groupParselet(p *Parser, leftParen Token) (expr Node, err feedback.Message) {
	var n Node
	var inner Expr
	// var rightParen Token

	if n, err = p.parseExpression(0); err != nil {
		return nil, err
	} else {
		var ok bool

		if inner, ok = n.(Expr); ok == false {
			return nil, feedback.Error{
				Classification: feedback.IllegalStatementError,
				File:           p.Lexer.Scanner.File,
				What: feedback.Selection{
					Description: "Expected an expression",
					Span:        source.Span{n.Pos(), n.End()},
				},
			}
		}
	}

	if _, err = p.Lexer.ExpectNext(RParenSymbol); err != nil {
		return nil, err
	}

	return inner, nil
}

func binaryInfixParselet(precedence int) binaryParselet {
	return func(p *Parser, tok Token, left Node) (expr Node, msg feedback.Message) {
		var right Node

		if right, msg = p.parseExpression(precedence); msg != nil {
			return nil, msg
		}

		var leftExpr Expr
		var rightExpr Expr
		var ok bool

		if leftExpr, ok = left.(Expr); ok == false {
			return nil, feedback.Error{
				Classification: feedback.IllegalStatementError,
				File:           p.Lexer.Scanner.File,
				What: feedback.Selection{
					Description: fmt.Sprintf(
						"expected left hand side to be expression, not `%T`",
						left),
					Span: source.Span{Start: left.Pos(), End: left.End()},
				},
			}
		}

		if rightExpr, ok = right.(Expr); ok == false {
			return nil, feedback.Error{
				Classification: feedback.IllegalStatementError,
				File:           p.Lexer.Scanner.File,
				What: feedback.Selection{
					Description: fmt.Sprintf(
						"expected right hand side to be expression, not `%T`",
						right),
					Span: source.Span{Start: right.Pos(), End: right.End()},
				},
			}
		}

		return &BinaryExpr{
			Operator: tok.Symbol,
			Left:     leftExpr,
			Right:    rightExpr,
		}, nil
	}
}

func indexAccessParselet(p *Parser, leftBracket Token, left Node) (expr Node, msg feedback.Message) {
	indexAccess := &IndexAccessExpr{
		LeftBracket: leftBracket,
	}
	var ok bool

	// Ensure that the left-hand side is an expression
	if indexAccess.Root, ok = left.(Expr); ok == false {
		return nil, feedback.Error{
			Classification: feedback.IllegalStatementError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: "expected left-hand side to be an expression",
				Span:        source.Span{left.Pos(), left.End()},
			},
		}
	}

	if node, msg := p.parseExpression(0); msg != nil {
		return nil, msg
	} else {
		if indexAccess.Index, ok = node.(Expr); ok == false {
			return nil, feedback.Error{
				Classification: feedback.IllegalStatementError,
				File:           p.Lexer.Scanner.File,
				What: feedback.Selection{
					Description: "expected index to be an expression",
					Span:        source.Span{node.Pos(), node.End()},
				},
			}
		}
	}

	if indexAccess.RightBracket, msg = p.Lexer.ExpectNext(RBracketSymbol); msg != nil {
		return nil, msg
	}

	return indexAccess, nil
}

func letDeclarationParselet(p *Parser, letKeyword Token) (expr Node, msg feedback.Message) {
	var name *IdentExpr
	var assignment Expr
	var left Node
	var right Node

	// precedence must be higher than assignment statement precedence so that
	// the identifier binds to the declaration, not as an assignment statement
	if left, msg = p.parseExpression(100); msg != nil {
		return nil, msg
	}

	if n, ok := left.(*IdentExpr); ok {
		name = n
	} else {
		return nil, feedback.Error{
			Classification: feedback.SyntaxError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: "Left hand of declaration must be an identifier",
				Span:        source.Span{Start: left.Pos(), End: left.End()},
			},
		}
	}

	if _, msg = p.Lexer.ExpectNext(TokenSymbol(":=")); msg != nil {
		return nil, msg
	}

	if right, msg = p.parseExpression(0); msg != nil {
		return nil, msg
	}

	if n, ok := right.(Expr); ok {
		assignment = n
	} else {
		return nil, feedback.Error{
			Classification: feedback.SyntaxError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("Right hand of declaration must be an expression"),
				Span:        source.Span{Start: right.Pos(), End: right.End()},
			},
		}
	}

	return &DeclarationStmt{
		LetKeyword: letKeyword,
		Assignee:   name,
		Assignment: assignment,
	}, nil
}

func assignmentParselet(p *Parser, colonEqual Token, left Node) (expr Node, msg feedback.Message) {
	var name *IdentExpr
	var assignment Expr

	if n, ok := left.(*IdentExpr); ok {
		name = n
	} else {
		return nil, feedback.Error{
			Classification: feedback.SyntaxError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("Left hand of assignment must be an identifier"),
				Span:        source.Span{Start: left.Pos(), End: left.End()},
			},
		}
	}

	right, msg := p.parseExpression(9)

	if msg != nil {
		return nil, msg
	}

	if n, ok := right.(Expr); ok {
		assignment = n
	} else {
		return nil, feedback.Error{
			Classification: feedback.SyntaxError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("Right hand of assignment must be an expression"),
				Span:        source.Span{Start: right.Pos(), End: right.End()},
			},
		}
	}

	return &AssignmentStmt{
		Assignee:   name,
		Assignment: assignment,
	}, nil
}

func dispatchParselet(p *Parser, leftParen Token, left Node) (expr Node, msg feedback.Message) {
	var root Expr
	var args []Expr

	if n, ok := left.(Expr); ok {
		root = n
	} else {
		return nil, feedback.Error{
			Classification: feedback.SyntaxError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("Expected expression to be called"),
				Span:        source.Span{Start: left.Pos(), End: left.End()},
			},
		}
	}

	for p.Lexer.PeekMatches(TokenSymbol(")")) == false {
		var arg Node
		var argExpr Expr
		var ok bool

		if arg, msg = p.parseExpression(0); msg != nil {
			return nil, msg
		}

		if argExpr, ok = arg.(Expr); ok == false {
			return nil, feedback.Error{
				Classification: feedback.IllegalStatementError,
				File:           p.Lexer.Scanner.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("Parameters must be expressions"),
					Span:        source.Span{Start: arg.Pos(), End: arg.End()},
				},
			}
		}

		// add argument to the list of arguments
		args = append(args, argExpr)

		if p.Lexer.PeekMatches(TokenSymbol(",")) {
			// consume comma after argument expression
			if _, msg = p.Lexer.ExpectNext(TokenSymbol(",")); msg != nil {
				return nil, msg
			}
		} else {
			// break the loop if no comma follows the argument
			break
		}
	}

	var rightParen Token

	if rightParen, msg = p.Lexer.ExpectNext(TokenSymbol(")")); msg != nil {
		return nil, msg
	}

	return &DispatchExpr{
		Root:       root,
		Arguments:  args,
		LeftParen:  leftParen,
		RightParen: rightParen,
	}, nil
}

func funcParselet(p *Parser, fnKeyword Token) (expr Node, err feedback.Message) {
	funcLiteral := &FuncLiteral{
		FnKeyword:        fnKeyword,
		Parameters:       []*Parameter{},
		ReturnAnnotation: nil,
		Body:             nil,
	}

	// Parse the function's parameter list's left-paren
	if funcLiteral.LeftParen, err = p.Lexer.ExpectNext(LParenSymbol); err != nil {
		return nil, err
	}

	if p.Lexer.PeekMatches(RParenSymbol) == false {
		// Parse the parameters
		for {
			var ident *IdentExpr
			var annotation TypeAnnotation = nil

			if tok, err := p.Lexer.ExpectNext(IdentSymbol); err != nil {
				return nil, err
			} else {
				ident = &IdentExpr{
					NamePos: tok.Span.Start,
					Name:    tok.Lexeme,
				}
			}

			// Parameter has a type annotation
			if p.Lexer.PeekMatches(ColonSymbol) {
				// Consume the colon
				p.Lexer.ExpectNext(ColonSymbol)

				// Parse the type annotation
				if annotation, err = typeAnnotationParselet(p); err != nil {
					return nil, err
				}
			}

			// Bind the parameter identifier and the optional type annotation
			// into a Field and append that Field to the function's FieldList.
			// Annotation is <nil> if no annotation was given
			funcLiteral.Parameters = append(funcLiteral.Parameters, &Parameter{
				Name:       ident,
				Annotation: annotation,
			})

			// Exit the loop if the parameter isn't followed by a comma
			if p.Lexer.PeekMatches(CommaSymbol) == false {
				break
			} else {
				// Otherwise consume the comma and then parse the next parameter
				p.Lexer.ExpectNext(CommaSymbol)
				continue
			}
		}
	}

	// Consume the closing right-paren on the parameter list
	if funcLiteral.RightParen, err = p.Lexer.ExpectNext(RParenSymbol); err != nil {
		return nil, err
	}

	// Check if the parameter list is followed by a function return-type annotation
	if p.Lexer.PeekMatches(ColonSymbol) {
		p.Lexer.ExpectNext(ColonSymbol)

		// Parse the return-type annotation
		if funcLiteral.ReturnAnnotation, err = typeAnnotationParselet(p); err != nil {
			return nil, err
		}
	}

	// Finally parse the function body (a series of 0 or more statements
	// contained between curly braces)
	if funcLiteral.Body, err = funcBodyParselet(p); err != nil {
		return nil, err
	}

	return funcLiteral, nil
}

func listParselet(p *Parser, lBracket Token) (expr Node, err feedback.Message) {
	var elements []Expr
	var rBracket Token

	// Consume any inner expression elements
	for {
		if p.Lexer.PeekMatches(RBracketSymbol) {
			break
		}

		if node, err := p.parseExpression(0); err != nil {
			return nil, err
		} else {
			if elem, ok := node.(Expr); ok == false {
				return nil, feedback.Error{
					Classification: feedback.IllegalStatementError,
					File:           p.Lexer.Scanner.File,
					What: feedback.Selection{
						Description: "Expected an expression",
						Span:        source.Span{Start: node.Pos(), End: node.End()},
					},
				}
			} else {
				elements = append(elements, elem)
			}
		}

		// Parser expects a comma after the expression in order to continue
		// parsing elements. Otherwise the loop breaks and will expect a closing
		// right-bracket
		if p.Lexer.PeekMatches(CommaSymbol) {
			p.Lexer.ExpectNext(CommaSymbol)
			continue
		}

		break
	}

	// Consume the right bracket closing the expression
	if rBracket, err = p.Lexer.ExpectNext(RBracketSymbol); err != nil {
		return nil, err
	}

	return &ListLiteral{
		LeftBracket:  lBracket,
		Elements:     elements,
		RightBracket: rBracket,
	}, nil
}

// parseFuncBody returns a FuncBody struct representing the collection
// of statements between braces in a function body
func funcBodyParselet(p *Parser) (body *FuncBody, msg feedback.Message) {
	var lBrace Token
	var stmts []Stmt
	var rBrace Token

	if lBrace, msg = p.Lexer.ExpectNext(TokenSymbol("{")); msg != nil {
		return nil, msg
	}

	if stmts, msg = p.parseStatementsUntil(func(tok Token) bool { return tok.Symbol == TokenSymbol("}") }); msg != nil {
		return nil, msg
	}

	if rBrace, msg = p.Lexer.ExpectNext(TokenSymbol("}")); msg != nil {
		return nil, msg
	}

	return &FuncBody{
		Statements: stmts,
		LeftBrace:  lBrace,
		RightBrace: rBrace,
	}, nil
}

func typeAnnotationParselet(p *Parser) (annotation TypeAnnotation, msg feedback.Message) {
	var leftParen, rightParen Token
	var params []TypeAnnotation
	var returnAnnotation TypeAnnotation

	if p.Lexer.PeekMatches(LBracketSymbol) {
		anno := ListTypeAnnotation{}

		// Consume the left-bracket
		if anno.LeftBracket, msg = p.Lexer.ExpectNext(LBracketSymbol); msg != nil {
			return nil, msg
		}

		// Consume inner annotation
		if anno.ElementType, msg = typeAnnotationParselet(p); msg != nil {
			return nil, msg
		}

		if anno.RightBracket, msg = p.Lexer.ExpectNext(RBracketSymbol); msg != nil {
			return nil, msg
		}

		return anno, nil
	} else if p.Lexer.PeekMatches(TokenSymbol("(")) {
		// Consume the left-paren
		if leftParen, msg = p.Lexer.ExpectNext(TokenSymbol("(")); msg != nil {
			return nil, msg
		}

		// Consume inner type annotation(s)
		for {
			// Catches empty parentheses
			if p.Lexer.PeekMatches(TokenSymbol(")")) {
				break
			}

			var anno TypeAnnotation

			// Parse the next interior annotation
			if anno, msg = typeAnnotationParselet(p); msg != nil {
				return nil, msg
			}

			// Add the newly parsed annotation to the list of parameters
			params = append(params, anno)

			// If the annotation is followed by a comma, then there are sill are
			// more annotations in the group
			if p.Lexer.PeekMatches(TokenSymbol(",")) {
				if _, msg := p.Lexer.ExpectNext(TokenSymbol(",")); msg != nil {
					return nil, msg
				}
			} else {
				// No comma after the last annotation, exit the loop so the
				// right-paren can be consumed
				break
			}
		}

		if rightParen, msg = p.Lexer.ExpectNext(TokenSymbol(")")); msg != nil {
			return nil, msg
		}
	} else if p.Lexer.PeekMatches(IdentSymbol) {
		// Consume a type-identifier and build a named type annotation
		namedType := NamedTypeAnnotation{}

		// Parse the IdentSymbol token into an *IdentExpr to use in the annotation
		if tok, msg := p.Lexer.ExpectNext(IdentSymbol); msg != nil {
			return nil, msg
		} else if node, msg := identParselet(p, tok); msg != nil {
			return nil, msg
		} else {
			namedType.Name = node.(*IdentExpr)
		}

		// Add the new annotation to the list of params if the next token is
		// a function token: `=>`. Otherwise just return the annotation
		if p.Lexer.PeekMatches(TokenSymbol("=>")) {
			params = append(params, namedType)
		} else {
			return namedType, nil
		}
	} else {
		tok, _ := p.Lexer.Next()

		return nil, feedback.Error{
			Classification: feedback.TypeAnnotationError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: "Missing parameters before fat arrow",
				Verbose:     "If function has no parameters use `()`. If there are no parameter constraints, use `Any`",
				Span:        tok.Span,
			},
		}
	}

	// If the parselet makes it this far, the type annotation must be a function
	// annotation so expect a far arrow token
	if _, msg = p.Lexer.ExpectNext(TokenSymbol("=>")); msg != nil {
		return nil, msg
	}

	if p.Lexer.PeekMatches(TokenSymbol("(")) || p.Lexer.PeekMatches(IdentSymbol) {
		if returnAnnotation, msg = typeAnnotationParselet(p); msg != nil {
			return nil, msg
		}
	} else {
		tok, _ := p.Lexer.Next()

		return nil, feedback.Error{
			Classification: feedback.TypeAnnotationError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: "Missing return type after fat arrow",
				Verbose:     "If function returns nothing use `None`. If the type is unknown use `Any`",
				Span:        tok.Span,
			},
		}
	}

	// Check if the annotation has already been
	return FuncTypeAnnotation{
		LeftParen:  leftParen,
		Parameters: params,
		RightParen: rightParen,
		ReturnType: returnAnnotation,
	}, nil
}

func printStatementParselet(p *Parser, printKeyword Token) (expr Node, msg feedback.Message) {
	var args []Expr

	// TODO support no arguments (just prints a newline) and multiple arguments
	if node, msg := p.parseExpression(0); msg == nil {
		if e, ok := node.(Expr); ok {
			// argument IS an expression
			args = append(args, e)
		} else {
			return nil, feedback.Error{
				Classification: feedback.SyntaxError,
				File:           p.Lexer.Scanner.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("Expected an expression"),
					Span:        source.Span{Start: node.Pos(), End: node.End()},
				},
			}
		}
	} else {
		return nil, msg
	}

	return &PrintStmt{
		PrintKeyword: printKeyword,
		Arguments:    args,
	}, nil
}

func returnStatementParselet(p *Parser, returnKeyword Token) (expr Node, msg feedback.Message) {
	var arg Expr

	// if the "return" keyword is followed by a semicolon, then it isn't
	// returning any values, otherwise collect any expressions following the
	// keyword as return values
	if p.Lexer.PeekMatches(TokenSymbol(";")) == false {
		if node, msg := p.parseExpression(0); msg == nil {
			var ok bool

			if arg, ok = node.(Expr); ok == false {
				return nil, feedback.Error{
					Classification: feedback.IllegalStatementError,
					File:           p.Lexer.Scanner.File,
					What: feedback.Selection{
						Description: fmt.Sprintf("Expected an expression"),
						Span:        source.Span{Start: node.Pos(), End: node.End()},
					},
				}
			}
		} else {
			return nil, msg
		}
	}

	return &ReturnStmt{
		ReturnKeyword: returnKeyword,
		Argument:      arg,
	}, nil
}

func ifStatementParselet(p *Parser, ifKeyword Token) (expr Node, msg feedback.Message) {
	ifClause := &Clause{
		Keyword: ifKeyword,
		Body:    &ClauseBody{},
	}

	ifStmt := &IfStmt{
		IfClause: ifClause,
	}

	// Parse the test condition, is returned as a statement
	if node, msg := p.parseExpression(0); msg != nil {
		return nil, msg
	} else {
		var ok bool

		// Try to convert the condition node to an expression, emit an error if the
		// conversion fails
		if ifClause.Condition, ok = node.(Expr); ok == false {
			// Condition is a statement, not an expression
			return nil, feedback.Error{
				Classification: feedback.IllegalStatementError,
				File:           p.Lexer.Scanner.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("Expected a conditional expression"),
					Span:        source.Span{Start: node.Pos(), End: node.End()},
				},
			}
		}
	}

	// Consume the colon after the test condition, before the body statements
	if ifClause.Body.Colon, msg = p.Lexer.ExpectNext(ColonSymbol); msg != nil {
		return nil, msg
	}

	endIfClauseTest := func(tok Token) bool {
		return tok.Symbol == TokenSymbol("elif") ||
			tok.Symbol == TokenSymbol("else") ||
			tok.Symbol == TokenSymbol("end")
	}

	// Consume the statements in the clause body
	if ifClause.Body.Statements, msg = p.parseStatementsUntil(endIfClauseTest); msg != nil {
		return nil, msg
	}

	// Parse any `elif` clauses
	if p.Lexer.PeekMatches(TokenSymbol("elif")) {
		for {
			elifClause := &Clause{
				Body: &ClauseBody{},
			}

			// Parse the `elif` keyword at the start of the clause
			if elifClause.Keyword, msg = p.Lexer.ExpectNext(TokenSymbol("elif")); msg != nil {
				return nil, msg
			}

			// Parse the test condition
			if node, msg := p.parseExpression(0); msg != nil {
				return nil, msg
			} else {
				var ok bool

				// Try to convert the condition node to an expression, emit an error if the
				// conversion fails
				if elifClause.Condition, ok = node.(Expr); ok == false {
					// Condition is a statement, not an expression
					return nil, feedback.Error{
						Classification: feedback.IllegalStatementError,
						File:           p.Lexer.Scanner.File,
						What: feedback.Selection{
							Description: fmt.Sprintf("Expected a conditional expression"),
							Span:        source.Span{Start: node.Pos(), End: node.End()},
						},
					}
				}
			}

			// Consume the colon after the test condition, before the body statements
			if elifClause.Body.Colon, msg = p.Lexer.ExpectNext(ColonSymbol); msg != nil {
				return nil, msg
			}

			endElifClauseTest := func(tok Token) bool {
				return tok.Symbol == TokenSymbol("elif") ||
					tok.Symbol == TokenSymbol("else") ||
					tok.Symbol == TokenSymbol("end")
			}

			// Consume the statements in the clause body until one of the
			// possible terminator keywords are reached
			if elifClause.Body.Statements, msg = p.parseStatementsUntil(endElifClauseTest); msg != nil {
				return nil, msg
			}

			// Append `elif` clause to the list of `elif` clauses
			ifStmt.ElifClauses = append(ifStmt.ElifClauses, elifClause)

			if p.Lexer.PeekMatches(TokenSymbol("elif")) {
				continue
			} else if p.Lexer.PeekMatches(TokenSymbol("else")) || p.Lexer.PeekMatches(TokenSymbol("end")) {
				break
			} else {
				tok, _ := p.Lexer.Next()

				// Next token was not an `end` keyword or an `elif` keyword
				// as expected so emit an error
				return nil, feedback.Error{
					Classification: feedback.SyntaxError,
					File:           p.Lexer.Scanner.File,
					What: feedback.Selection{
						Description: "Unexpected keyword",
						Span:        tok.Span,
					},
				}
			}
		}
	}

	// Parse an optional `else` clause
	if p.Lexer.PeekMatches(TokenSymbol("else")) {
		elseClause := &Clause{
			Body: &ClauseBody{},
		}

		// Parse the `else` keyword at the start of the clause
		if elseClause.Keyword, msg = p.Lexer.ExpectNext(TokenSymbol("else")); msg != nil {
			return nil, msg
		}

		// Consume the colon after the `else` keyword, before the clause statements
		if elseClause.Body.Colon, msg = p.Lexer.ExpectNext(ColonSymbol); msg != nil {
			return nil, msg
		}

		endElseClauseTest := func(tok Token) bool {
			return tok.Symbol == TokenSymbol("end")
		}

		if elseClause.Body.Statements, msg = p.parseStatementsUntil(endElseClauseTest); msg != nil {
			return nil, msg
		}

		ifStmt.ElseClause = elseClause
	}

	// Consume the `end` keyword at the end of the `if` statement
	if ifStmt.EndKeyword, msg = p.Lexer.ExpectNext(TokenSymbol("end")); msg != nil {
		return nil, msg
	}

	return ifStmt, nil
}

func loopStatementParselet(p *Parser, loopKeyword Token) (expr Node, msg feedback.Message) {
	clause := &Clause{
		Keyword: loopKeyword,
		Body:    &ClauseBody{},
	}

	stmt := &LoopStmt{
		Clause: clause,
	}

	if node, msg := p.parseExpression(0); msg != nil {
		return nil, msg
	} else {
		var ok bool

		if clause.Condition, ok = node.(Expr); ok == false {
			return nil, feedback.Error{
				Classification: feedback.IllegalStatementError,
				File:           p.Lexer.Scanner.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("Expected a condiional expression, found a statement"),
					Span:        source.Span{Start: node.Pos(), End: node.End()},
				},
			}
		}
	}

	// Consume the colon after the test condition, before the body statements
	if clause.Body.Colon, msg = p.Lexer.ExpectNext(ColonSymbol); msg != nil {
		return nil, msg
	}

	endBodyTest := func(tok Token) bool {
		return tok.Symbol == TokenSymbol("end")
	}

	// Consume the statements in the clause body
	if clause.Body.Statements, msg = p.parseStatementsUntil(endBodyTest); msg != nil {
		return nil, msg
	}

	// Consume the `end` keyword after the clause body
	if stmt.EndKeyword, msg = p.Lexer.ExpectNext(TokenSymbol("end")); msg != nil {
		return nil, msg
	}

	return stmt, nil
}

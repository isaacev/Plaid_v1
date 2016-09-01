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

		return &IntegerExpr{
			Lexeme: tok.Lexeme,
			Value:  i32,
			Start:  tok.Span.Start,
		}, nil
	case DecimalSymbol:
		// FIXME: parse more than just floats and handle this error
		f64, _ := strconv.ParseFloat(tok.Lexeme, precisionBits)
		f32 := float32(f64)

		return &DecimalExpr{
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

		return &StringExpr{
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

func leadingParenParselet(p *Parser, lParen Token) (expr Node, msg feedback.Message) {
	var rParen Token

	if p.Lexer.PeekMatches(TokenSymbol(")")) {
		// empty list of parameters
		if rParen, msg = p.Lexer.ExpectNext(TokenSymbol(")")); msg != nil {
			return nil, msg
		}

		return &FieldList{
			LeftParen:  lParen,
			RightParen: rParen,
		}, nil
	}

	var elements []Node
	noTrailingComma := false

	for {
		if noTrailingComma {
			if rParen, msg = p.Lexer.ExpectNext(TokenSymbol(")")); msg != nil {
				return nil, msg
			}

			if p.Lexer.PeekMatches(TokenSymbol("=>")) {
				// group is a field list, not an expression group
				var fields []*TypeAnnotationStmt

				// convert expression nodes to TypeAnnotationStmt's, throw an
				// error if none are valid TypeAnnotationStmt's or IdentExpr's
				for _, generic := range elements {
					switch node := generic.(type) {
					case *IdentExpr:
						fields = append(fields, &TypeAnnotationStmt{
							Identifier:   node,
							ExplicitType: false,
						})
					case *TypeAnnotationStmt:
						fields = append(fields, node)
					default:
						return nil, feedback.Error{
							Classification: feedback.SyntaxError,
							File:           p.Lexer.Scanner.File,
							What: feedback.Selection{
								Description: fmt.Sprintf(
									"Unexpected expression `%T` in field list",
									node),
								Span: source.Span{Start: node.Pos(), End: node.End()},
							},
						}
					}
				}

				return &FieldList{
					Fields:     fields,
					LeftParen:  lParen,
					RightParen: rParen,
				}, nil
			} else if len(elements) == 1 {
				// if parenthetical has 1 element, that element is an expression
				// and the parenthetical is NOT followed by a far arrow, asssume
				// the parenthetical is a grouped expression
				if group, ok := elements[0].(Expr); ok {
					return group, msg
				}
			}
		}

		var elem Node

		if elem, msg = p.parseExpression(0); msg != nil {
			return nil, msg
		}

		elements = append(elements, elem)

		if p.Lexer.PeekMatches(TokenSymbol(",")) {
			if _, msg := p.Lexer.ExpectNext(TokenSymbol(",")); msg != nil {
				return nil, msg
			}
		} else {
			noTrailingComma = true
		}
	}
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
				Classification: feedback.SyntaxError,
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
				Classification: feedback.SyntaxError,
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

func typeAssociationParselet(p *Parser, doubleColon Token, left Node) (expr Node, msg feedback.Message) {
	var name *IdentExpr
	var annotation *IdentExpr

	if n, ok := left.(*IdentExpr); ok {
		name = n
	} else {
		return nil, feedback.Error{
			Classification: feedback.SyntaxError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("Left hand of type annotation must be an identifier"),
				Span:        source.Span{Start: left.Pos(), End: left.End()},
			},
		}
	}

	right, msg := p.parseExpression(0)

	if msg != nil {
		return nil, msg
	}

	if n, ok := right.(*IdentExpr); ok {
		annotation = n
	} else {
		return nil, feedback.Error{
			Classification: feedback.SyntaxError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("Right hand of type annotation must be a valid type identifier"),
				Span:        source.Span{Start: right.Pos(), End: right.End()},
			},
		}
	}

	return &TypeAnnotationStmt{
		Identifier:   name,
		Annotation:   annotation,
		ExplicitType: true,
	}, nil
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

	right, msg := p.parseExpression(0)

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
	var args []Expr
	var root *IdentExpr

	if n, ok := left.(*IdentExpr); ok {
		root = n
	} else {
		return nil, feedback.Error{
			Classification: feedback.SyntaxError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("Expected identifier in function dispatch"),
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
				Classification: feedback.SyntaxError,
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

func functionParselet(p *Parser, fatArrow Token, group Node) (expr Node, msg feedback.Message) {
	var params *FieldList
	var body *FunctionBody
	var ok bool

	if params, ok = group.(*FieldList); ok == false {
		return nil, feedback.Error{
			Classification: feedback.SyntaxError,
			File:           p.Lexer.Scanner.File,
			What: feedback.Selection{
				Description: "Expected field list",
				Span:        source.Span{Start: group.Pos(), End: group.End()},
			},
		}
	}

	if body, msg = p.parseFunctionBody(); msg != nil {
		return nil, msg
	}

	return &FuncExpr{
		Parameters: params,
		Body:       body,
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
	var args []Expr

	// if the "return" keyword is followed by a semicolon, then it isn't
	// returning any values, otherwise collect any expressions following the
	// keyword as return values
	if p.Lexer.PeekMatches(TokenSymbol(";")) == false {
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
	}

	return &ReturnStmt{
		ReturnKeyword: returnKeyword,
		Arguments:     args,
	}, nil
}

func ifStatementParselet(p *Parser, ifKeyword Token) (expr Node, msg feedback.Message) {
	var condition Expr
	var node Node

	if node, msg = p.parseExpression(0); msg == nil {
		if e, ok := node.(Expr); ok {
			// if-condition is an expression, not a statement
			condition = e
		} else {
			return nil, feedback.Error{
				Classification: feedback.SyntaxError,
				File:           p.Lexer.Scanner.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("Expected a conditional expression"),
					Span:        source.Span{Start: node.Pos(), End: node.End()},
				},
			}
		}
	} else {
		return nil, msg
	}

	var body *ConditionalBody

	if body, msg = p.parseConditionalBody(); msg != nil {
		return nil, msg
	}

	return &IfStmt{
		IfKeyword: ifKeyword,
		Condition: condition,
		Body:      body,
	}, nil
}

func loopStatementParselet(p *Parser, loopKeyword Token) (expr Node, msg feedback.Message) {
	var condition Expr
	var node Node

	if node, msg = p.parseExpression(0); msg == nil {
		if e, ok := node.(Expr); ok {
			// if-condition is an expression, not a statement
			condition = e
		} else {
			return nil, feedback.Error{
				Classification: feedback.SyntaxError,
				File:           p.Lexer.Scanner.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("Expected a conditional expression"),
					Span:        source.Span{Start: node.Pos(), End: node.End()},
				},
			}
		}
	} else {
		return nil, msg
	}

	var body *ConditionalBody

	if body, msg = p.parseConditionalBody(); msg != nil {
		return nil, msg
	}

	return &LoopStmt{
		LoopKeyword: loopKeyword,
		Condition:   condition,
		Body:        body,
	}, nil
}

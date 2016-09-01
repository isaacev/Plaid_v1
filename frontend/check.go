package frontend

import (
	"fmt"

	"github.com/isaacev/Plaid/feedback"
	"github.com/isaacev/Plaid/source"
)

// Check takes a Program AST node and validates the type properties of the
// entire tree. Any type errors encountered while crawling the AST are returned
func Check(file *source.File, prog *Program) (msgs []feedback.Message) {
	globalScope := newGlobalScope(file)

	for _, stmt := range prog.Statements {
		msgs = append(msgs, checkStatement(globalScope, stmt)...)
	}

	for i, varName := range globalScope.registeredVariables {
		prog.Locals = append(prog.Locals, &LocalRecord{
			Name:        varName,
			LookupIndex: i,
		})
	}

	return msgs
}

// checkStatement validates types within a single statement in the context of a
// given scope. Any expressions encountered are passed to the "checkExpression"
// function. If any type irregularities occur, they are emitted via the list
// of messages.
func checkStatement(scope *Scope, stmt Stmt) (msgs []feedback.Message) {
	switch s := stmt.(type) {
	case *TypeAnnotationStmt:
		msgs = checkTypeAnnotationStmt(scope, s)
	case *DeclarationStmt:
		msgs = checkDeclarationStmt(scope, s)
	case *AssignmentStmt:
		msgs = checkAssignmentStmt(scope, s)
	case *IfStmt:
		msgs = checkIfStmt(scope, s)
	case *LoopStmt:
		msgs = checkLoopStmt(scope, s)
	case *PrintStmt:
		msgs = checkPrintStmt(scope, s)
	case *ReturnStmt:
		msgs = checkReturnStmt(scope, s)
	case Expr:
		_, msgs = checkExpression(scope, s)
	default:
		panic(fmt.Sprintf("UNKNOWN STATEMENT NODE: %T\n", s))
	}

	return msgs
}

func checkTypeAnnotationStmt(scope *Scope, stmt *TypeAnnotationStmt) []feedback.Message {
	sig, msgs := signatureFromAnnotation(scope, stmt)
	scope.registerLocalVariable(stmt.Identifier.Name, sig)
	return msgs
}

func checkDeclarationStmt(scope *Scope, stmt *DeclarationStmt) []feedback.Message {
	sig, msgs := checkExpression(scope, stmt.Assignment)

	if assigneeSig := scope.lookupLocalVariable(stmt.Assignee.Name); assigneeSig == nil {
		// This is good, variable hasn't been declared yet
		// infer type from assignment
		if sig.Definition.Start.Line == 0 {
			// Signature hasn't been given a definition location span yet
			sig.Definition = source.Span{stmt.Pos(), stmt.End()}
		}

		scope.registerLocalVariable(stmt.Assignee.Name, sig)
	} else {
		// This is bad, variable already exists in scope, cannot be
		// re-declared
		msgs = append(msgs, feedback.Error{
			Classification: feedback.RedeclarationError,
			File:           scope.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("`%s` redeclared here",
					stmt.Assignee.Name),
				Span: source.Span{stmt.Assignee.Pos(), stmt.Assignee.End()},
			},
			Why: []feedback.Selection{
				{
					Description: fmt.Sprintf("`%s` originally declared here", stmt.Assignee.Name),
					Span:        assigneeSig.Definition,
				},
			},
		})
	}

	if funcExpr, ok := stmt.Assignment.(*FuncExpr); ok {
		msgs = append(msgs, checkFunctionBody(scope, sig, funcExpr)...)
	}

	return msgs
}

func checkAssignmentStmt(scope *Scope, stmt *AssignmentStmt) []feedback.Message {
	sig, msgs := checkExpression(scope, stmt.Assignment)

	if assigneeSig, isLocal := scope.lookupVariable(stmt.Assignee.Name); assigneeSig != nil {
		if isLocal == false {
			// Register assignee as an upvalue
			scope.registerUpvalue(stmt.Assignee.Name)
		}

		// environment has already been fed some info about this variable
		if assigneeSig.Output != sig.Output {
			// TODO: if sig.Output.isDescendantOf(assigneeSig.Output) is
			// true, try to coerce "stmt.Assignee" type to "sig.Output" type

			// Program attempting to assign a type to a variable of a
			// different type
			msgs = append(msgs, feedback.Error{
				Classification: feedback.RedeclarationError,
				File:           scope.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("variable `%s` (type `%s`) cannot be assigned type `%s`",
						stmt.Assignee.Name,
						assigneeSig.Output.Name,
						sig.Output.Name),
					Span: source.Span{stmt.Pos(), stmt.End()},
				},
				Why: []feedback.Selection{
					{
						Description: fmt.Sprintf("`%s` originally assigned type `%s` here", stmt.Assignee.Name, assigneeSig.Output.Name),
						Span:        assigneeSig.Definition,
					},
				},
			})
		}
	} else {
		// Emit an error if a variable is used before it has been declared
		msgs = append(msgs, feedback.Error{
			Classification: feedback.UndefinedVariableError,
			File:           scope.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("variable `%s` cannot be assigned before it has been declared",
					stmt.Assignee.Name),
				Span: source.Span{stmt.Pos(), stmt.End()},
			},
		})

		// Return a generic `Any` signature so that type checking can
		// continue without a fatally missing type
		sig = &Signature{
			Output:     scope.typeTable.AnyType,
			Definition: source.Span{stmt.Pos(), stmt.End()},
		}
	}

	if funcExpr, ok := stmt.Assignment.(*FuncExpr); ok {
		msgs = append(msgs, checkFunctionBody(scope, sig, funcExpr)...)
	}

	return msgs
}

func checkIfStmt(scope *Scope, stmt *IfStmt) []feedback.Message {
	conditionSig, msgs := checkExpression(scope, stmt.Condition)

	if conditionSig.Output != scope.typeTable.Table["Bool"] {
		msgs = append(msgs, feedback.Error{
			Classification: feedback.MismatchedTypeError,
			File:           scope.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("condition must have type `Bool`, instead found `%s`", conditionSig.Output.Name),
				Span:        source.Span{stmt.Condition.Pos(), stmt.Condition.End()},
			},
		})
	}

	msgs = append(msgs, checkConditionalBody(scope, stmt.Body)...)
	return msgs
}

func checkLoopStmt(scope *Scope, stmt *LoopStmt) []feedback.Message {
	conditionSig, msgs := checkExpression(scope, stmt.Condition)

	if conditionSig.Output != scope.typeTable.Table["Boolean"] {
		msgs = append(msgs, feedback.Error{
			Classification: feedback.MismatchedTypeError,
			File:           scope.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("condition must have type `Boolean`, instead found `%s`", conditionSig.Output.Name),
				Span:        source.Span{stmt.Condition.Pos(), stmt.Condition.End()},
			},
		})
	}

	msgs = append(msgs, checkConditionalBody(scope, stmt.Body)...)
	return msgs
}

func checkPrintStmt(scope *Scope, stmt *PrintStmt) []feedback.Message {
	var msgs []feedback.Message

	if len(stmt.Arguments) > 0 {
		// As long as print arguments are expressions, don't worry about
		// their types since the statment handles that internally, just
		// ensure that each expression is internally valid
		for _, arg := range stmt.Arguments {
			sig, moreMsgs := checkExpression(scope, arg)
			msgs = append(msgs, moreMsgs...)

			// Check the bodies of any anonymous functions being printed
			if funcExpr, ok := arg.(*FuncExpr); ok {
				msgs = append(msgs, checkFunctionBody(scope, sig, funcExpr)...)
			}
		}
	}

	return msgs
}

func checkReturnStmt(scope *Scope, stmt *ReturnStmt) (msgs []feedback.Message) {
		}
	}

	return msgs
}

// checkExpression validates types within a single expression in the context of
// a given scope. Any type irregularities are emitted via the list of messages
// returned
func checkExpression(scope *Scope, expr Expr) (sig *Signature, msgs []feedback.Message) {
	switch e := expr.(type) {
	case Literal:
		sig, msgs = checkLiteral(scope, e)
	case *IdentExpr:
		sig, msgs = checkIdentExpr(scope, e)
	case *BinaryExpr:
		sig, msgs = checkBinaryExpr(scope, e)
	case *DispatchExpr:
		sig, msgs = checkDispatchExpr(scope, e)
	case *FuncExpr:
		sig, msgs = checkFuncExpr(scope, e)
	default:
		panic(fmt.Sprintf("UNKNOWN AST NODE: %T\n", e))
	}

	return sig, msgs
}

func checkLiteral(scope *Scope, expr Literal) (sig *Signature, msgs []feedback.Message) {
	var t *Type

	switch literal := expr.(type) {
	case *IntegerExpr:
		t = scope.typeTable.Table["Int"]
		literal.t = t
	case *DecimalExpr:
		t = scope.typeTable.Table["Dec"]
		literal.t = t
	case *StringExpr:
		t = scope.typeTable.Table["Str"]
		literal.t = t
	}

	return &Signature{Output: t}, nil
}

func checkIdentExpr(scope *Scope, expr *IdentExpr) (sig *Signature, msgs []feedback.Message) {
	if sig, isLocal := scope.lookupVariable(expr.Name); sig == nil {
		err := feedback.Error{
			Classification: feedback.UndefinedVariableError,
			File:           scope.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("variable `%s` is undeclared", expr.Name),
				Span:        source.Span{expr.Pos(), expr.End()},
			},
		}

		// Return a generic `Any` signature so that type checking can
		// continue without a fatally missing type
		sig = &Signature{
			Output:     scope.typeTable.AnyType,
			Definition: source.Span{expr.Pos(), expr.End()},
		}

		return sig, append(msgs, err)
	} else if isLocal {
		// Variable is local and already declared, so tag it with its type
		expr.t = sig.Output
		return sig, nil
	} else {
		// Register identifier as an upvalue
		scope.registerUpvalue(expr.Name)
		return sig, nil
	}
}

func checkBinaryExpr(scope *Scope, expr *BinaryExpr) (sig *Signature, msgs []feedback.Message) {
	sig = &Signature{}

	var sigLeft *Signature
	var sigRight *Signature
	var moreMsgs []feedback.Message
	var op string
	var any *Type

	// Create some shortcuts for repeatedly used values
	op = string(expr.Operator)
	any = scope.typeTable.AnyType

	// Evaluate the type of the left operand
	sigLeft, moreMsgs = checkExpression(scope, expr.Left)
	msgs = append(msgs, moreMsgs...)

	// Evaluate the type of the right operand
	sigRight, moreMsgs = checkExpression(scope, expr.Right)
	msgs = append(msgs, moreMsgs...)

	if sigLeft.Output == any || sigRight.Output == any {
		if exists, result := any.hasMethod(op, any); exists {
			// Type `Any` has some comparison and logical methods declared because
			// the type of a binary `==` operation is always boolean so its type
			// can be known
			sig.Output = result
		} else {
			sig.Output = any
		}
	} else {
		// Both the left and right operands are statically typed, not `Any`
		if exists, result := sigLeft.Output.hasMethod(op, sigRight.Output); exists {
			sig.Output = result
		} else {
			msgs = append(msgs, feedback.Error{
				Classification: feedback.MismatchedTypeError,
				File:           scope.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("incompatible types `%s` and `%s`",
						sigLeft.Output.Name,
						sigRight.Output.Name),
					Span: source.Span{expr.Pos(), expr.End()},
				},
			})

			sig.Output = any
		}
	}

	expr.t = sig.Output
	return sig, nil
}

func checkDispatchExpr(scope *Scope, expr *DispatchExpr) (sig *Signature, msgs []feedback.Message) {
	sig = &Signature{}

	if funcSig, _ := scope.lookupVariable(expr.Root.Name); funcSig != nil {
		totalExpectedParams := len(funcSig.Inputs)
		totalGivenArgs := len(expr.Arguments)
		sig.Output = funcSig.Output

		if totalExpectedParams == totalGivenArgs {
			for n := 0; n < totalExpectedParams; n++ {
				nthParamType := funcSig.Inputs[n]
				nthArg := expr.Arguments[n]

				nthArgSignature, moreMsgs := checkExpression(scope, nthArg)
				msgs = append(msgs, moreMsgs...)

				// If the parameter can take any type, the rest of the analysis
				// in this loop can be skipped
				if nthParamType == scope.typeTable.AnyType {
					continue
				}

				if nthParamType != nthArgSignature.Output {
					msgs = append(msgs, feedback.Error{
						Classification: feedback.MismatchedTypeError,
						File:           scope.File,
						What: feedback.Selection{
							Description: fmt.Sprintf("`%s` cannot be used as type `%s`",
								nthArgSignature.Output.Name,
								nthParamType.Name),
							Span: source.Span{nthArg.Pos(), nthArg.End()},
						},
						Why: []feedback.Selection{
							{
								Description: fmt.Sprintf("type `%s` expected as %s argument",
									nthParamType.Name,
									toOrdinal(n+1)),
								Span: funcSig.InputDefinitions[n],
							},
						},
					})
				}
			}
		} else {
			msgs = append(msgs, feedback.Error{
				Classification: feedback.MismatchedArgumentsError,
				File:           scope.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("%d arguments passed", totalGivenArgs),
					Span: source.Span{
						Start: expr.LeftParen.Span.Start,
						End:   expr.RightParen.Span.End,
					},
				},
				Why: []feedback.Selection{
					{
						Description: fmt.Sprintf("`%s` expected %d arguments",
							expr.Root.Name,
							totalExpectedParams),
						Span: funcSig.Definition,
					},
				},
			})
		}
	} else {
		// Even though the function signature is unknown, check each
		// argument for internal type validation
		for n := 0; n < len(expr.Arguments); n++ {
			nthArg := expr.Arguments[n]
			_, moreMsgs := checkExpression(scope, nthArg)
			msgs = append(msgs, moreMsgs...)
		}

		// Give dispatch expression a Root Type, even though lookup failed
		sig.Output = scope.typeTable.AnyType

		msgs = append(msgs, feedback.Error{
			Classification: feedback.UndefinedVariableError,
			File:           scope.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("unrecognized function `%s`", expr.Root.Name),
				Span:        source.Span{expr.Pos(), expr.End()},
			},
		})
	}

	return sig, msgs
}

// checkFuncExpr is only responsible for analyzing the type-signature of a
// function, not for doing further type analysis on the body of the function.
// That responsibility falls to the `checkFunctionBody` function
func checkFuncExpr(scope *Scope, expr *FuncExpr) (sig *Signature, msgs []feedback.Message) {
	sig = &Signature{
		Output: scope.typeTable.AnyType,
		Definition: source.Span{
			Start: expr.Parameters.LeftParen.Span.Start,
			End:   expr.Parameters.RightParen.Span.End,
		},
	}

	for _, param := range expr.Parameters.Fields {
		var paramSig *Signature
		var moreMsgs []feedback.Message

		paramSig, moreMsgs = signatureFromAnnotation(scope, param)
		msgs = append(msgs, moreMsgs...)

		sig.Inputs = append(sig.Inputs, paramSig.Output)
		sig.InputDefinitions = append(sig.InputDefinitions, source.Span{param.Pos(), param.End()})
	}

	expr.t = sig.Output
	return sig, msgs
}

// TODO should return a Type struct that represents the proven return type of
// the function body
func checkFunctionBody(scope *Scope, sig *Signature, expr *FuncExpr) (msgs []feedback.Message) {
	subScope := scope.subScope()
	paramNames := make(map[string]bool)

	for i, param := range expr.Parameters.Fields {
		paramName := param.Identifier.Name
		paramNames[paramName] = true
		paramSig := &Signature{
			Output:     sig.Inputs[i],
			Definition: source.Span{param.Pos(), param.End()},
		}
		subScope.registerLocalVariable(paramName, paramSig)
	}

	for _, stmt := range expr.Body.Statements {
		msgs = append(msgs, checkStatement(subScope, stmt)...)
	}

	for i, varName := range subScope.registeredVariables {
		expr.Locals = append(expr.Locals, &LocalRecord{
			Name:        varName,
			IsParameter: paramNames[varName],
			LookupIndex: i,
		})
	}

	for _, record := range subScope.upvalues {
		expr.Upvalues = append(expr.Upvalues, record)
	}

	return msgs
}

// checkConditionalBody checks the statements inside the body of a conditional
// statement
func checkConditionalBody(scope *Scope, body *ConditionalBody) (msgs []feedback.Message) {
	for _, stmt := range body.Statements {
		msgs = append(msgs, checkStatement(scope, stmt)...)
	}

	return msgs
}

// signatureFromAnnotation computes the type of a given TypeAnnotationStmt
func signatureFromAnnotation(scope *Scope, s *TypeAnnotationStmt) (sig *Signature, msgs []feedback.Message) {
	var t *Type

	if s.ExplicitType {
		var ok bool

		if t, ok = scope.typeTable.Table[s.Annotation.Name]; ok == false {
			// Even though the annotation references an undefined type, populate
			// the Scope's type table and type definition table with the
			// undefined type's data so that later references to the type will
			// have more useful error messages
			t = &Type{
				Name:       s.Annotation.Name,
				Definition: source.Span{s.Annotation.Pos(), s.Annotation.End()},
			}

			scope.typeTable.Table[s.Annotation.Name] = t

			msgs = append(msgs, feedback.Error{
				Classification: feedback.UndefinedTypeError,
				File:           scope.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("type `%s` has not been defined", s.Annotation.Name),
					Span:        t.Definition,
				},
			})
		}
	} else {
		t = scope.typeTable.AnyType
	}

	return &Signature{
		Output:     t,
		Definition: source.Span{s.Pos(), s.End()},
	}, msgs
}

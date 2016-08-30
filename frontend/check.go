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
	sig := &Signature{}

	switch s := stmt.(type) {
	case *TypeAnnotationStmt:
		sig, msgs = signatureFromAnnotation(scope, s)
		scope.registerLocalVariable(s.Identifier.Name, sig)
	case *DeclarationStmt:
		sig, msgs = checkExpression(scope, s.Assignment)

		if assigneeSig := scope.lookupLocalVariable(s.Assignee.Name); assigneeSig == nil {
			// this is good, variable hasn't been declared yet
			// infer type from assignment
			sig.Definition = source.Span{Start: s.Pos(), End: s.End()}
			scope.registerLocalVariable(s.Assignee.Name, sig)
		} else {
			// this is bad, variable already exists in scope, cannot be
			// re-declared
			msgs = append(msgs, feedback.Error{
				Classification: feedback.TypeCheckError,
				File:           scope.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("variable `%s` has already been declared",
						s.Assignee.Name),
					Span: source.Span{Start: s.Assignment.Pos(), End: s.Assignment.End()},
				},
				Why: []feedback.Selection{
					{
						Description: fmt.Sprintf("`%s` originally declared here", s.Assignee.Name),
						Span:        assigneeSig.Definition,
					},
				},
			})
		}

		if funcExpr, ok := s.Assignment.(*FuncExpr); ok {
			msgs = append(msgs, checkFunctionBody(scope, sig, funcExpr)...)
		}
	case *AssignmentStmt:
		sig, msgs = checkExpression(scope, s.Assignment)

		if assigneeSig, isLocal := scope.lookupVariable(s.Assignee.Name); assigneeSig != nil {
			if isLocal == false {
				// register assignee as an upvalue
				scope.registerUpvalue(s.Assignee.Name)
			}

			// environment has already been fed some info about this variable
			if assigneeSig.Output != sig.Output {
				// TODO: if sig.Output.isDescendantOf(assigneeSig.Output) is
				// true, try to coerce "s.Assignee" type to "sig.Output" type

				// program attempting to assign a type to a variable of a
				// different type
				msgs = append(msgs, feedback.Error{
					Classification: feedback.TypeCheckError,
					File:           scope.File,
					What: feedback.Selection{
						Description: fmt.Sprintf("variable `%s` (type `%s`) cannot be assigned type `%s`",
							s.Assignee.Name,
							assigneeSig.Output.Name,
							sig.Output.Name),
						Span: source.Span{Start: s.Assignment.Pos(), End: s.Assignment.End()},
					},
					Why: []feedback.Selection{
						{
							Description: fmt.Sprintf("`%s` originally assigned type `%s` here", s.Assignee.Name, assigneeSig.Output.Name),
							Span:        assigneeSig.Definition,
						},
					},
				})
			}
		} else {
			// emit an error if a variable is used before it has been declared
			msgs = append(msgs, feedback.Error{
				Classification: feedback.TypeCheckError,
				File:           scope.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("variable `%s` cannot be assigned before it has been declared",
						s.Assignee.Name),
					Span: source.Span{Start: s.Assignment.Pos(), End: s.Assignment.End()},
				},
			})

			// return a generic Object signature so that type checking can
			// continue without a fatally missing type
			sig = &Signature{
				Output:     scope.inheritanceTree.RootType,
				Definition: source.Span{Start: s.Pos(), End: s.End()},
			}
		}

		if funcExpr, ok := s.Assignment.(*FuncExpr); ok {
			msgs = append(msgs, checkFunctionBody(scope, sig, funcExpr)...)
		}
	case *IfStmt:
		var conditionSig *Signature
		conditionSig, msgs = checkExpression(scope, s.Condition)

		if conditionSig.Output != scope.inheritanceTree.Table["Boolean"] {
			msgs = append(msgs, feedback.Error{
				Classification: feedback.TypeCheckError,
				File:           scope.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("condition must have type `Boolean`, instead found `%s`", conditionSig.Output.Name),
					Span:        source.Span{Start: s.Condition.Pos(), End: s.Condition.End()},
				},
			})
		}

		msgs = append(msgs, checkConditionalBody(scope, s.Body)...)
	case *LoopStmt:
		var conditionSig *Signature
		conditionSig, msgs = checkExpression(scope, s.Condition)

		if conditionSig.Output != scope.inheritanceTree.Table["Boolean"] {
			msgs = append(msgs, feedback.Error{
				Classification: feedback.TypeCheckError,
				File:           scope.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("condition must have type `Boolean`, instead found `%s`", conditionSig.Output.Name),
					Span:        source.Span{Start: s.Condition.Pos(), End: s.Condition.End()},
				},
			})
		}

		msgs = append(msgs, checkConditionalBody(scope, s.Body)...)
	case *PrintStmt:
		if len(s.Arguments) > 0 {
			// as long as print arguments are expressions, don't worry about
			// their types since the statment handles that internally, just
			// ensure that each expression is internally valid
			for _, arg := range s.Arguments {
				sig, msgs = checkExpression(scope, arg)

				// check the bodies of any anonymous functions being printed
				if funcExpr, ok := arg.(*FuncExpr); ok {
					msgs = append(msgs, checkFunctionBody(scope, sig, funcExpr)...)
				}
			}
		}
	case *ReturnStmt:
		if len(s.Arguments) > 0 {
			for _, arg := range s.Arguments {
				sig, msgs = checkExpression(scope, arg)

				// check the bodies of any anonymous functions being return
				if funcExpr, ok := arg.(*FuncExpr); ok {
					msgs = append(msgs, checkFunctionBody(scope, sig, funcExpr)...)
				}
			}
		}
	case Expr:
		_, msgs = checkExpression(scope, s)
	default:
		fmt.Printf("UNKNOWN STATEMENT NODE: %T\n", s)
	}

	return msgs
}

// checkExpression validates types within a single expression in the context of
// a given scope. Any type irregularities are emitted via the list of messages
// returned
func checkExpression(scope *Scope, expr Expr) (sig *Signature, msgs []feedback.Message) {
	sig = &Signature{}
	t := &Type{}

	switch e := expr.(type) {
	case Literal:
		switch e.(type) {
		case *IntegerExpr:
			t = scope.inheritanceTree.Table["Integer"]
		case *DecimalExpr:
			t = scope.inheritanceTree.Table["Decimal"]
		case *StringExpr:
			t = scope.inheritanceTree.Table["String"]
		}

		sig.Output = t
	case *IdentExpr:
		var isLocal bool

		if sig, isLocal = scope.lookupVariable(e.Name); sig == nil {
			msgs = append(msgs, feedback.Error{
				Classification: feedback.TypeCheckError,
				File:           scope.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("variable `%s` is undeclared", e.Name),
					Span:        source.Span{Start: e.Pos(), End: e.End()},
				},
			})

			// return a generic Object signature so that type checking can
			// continue without a fatally missing type
			sig = &Signature{
				Output:     scope.inheritanceTree.RootType,
				Definition: source.Span{Start: e.Pos(), End: e.End()},
			}
		} else if isLocal == false {
			// register identifier as an upvalue
			scope.registerUpvalue(e.Name)
		}
	case *BinaryExpr:
		var sigLeft *Signature
		var sigRight *Signature
		var moreMsgs []feedback.Message

		sigLeft, moreMsgs = checkExpression(scope, e.Left)
		msgs = append(msgs, moreMsgs...)

		sigRight, moreMsgs = checkExpression(scope, e.Right)
		msgs = append(msgs, moreMsgs...)

		if sigLeft.Output != sigRight.Output {
			lca := scope.inheritanceTree.lowestCommonAncestor(sigLeft.Output, sigRight.Output)

			if lca == nil {
				msgs = append(msgs, feedback.Error{
					Classification: feedback.TypeCheckError,
					File:           scope.File,
					What: feedback.Selection{
						Description: fmt.Sprintf("incompatible types `%s` and `%s`", sigLeft.Output.Name, sigRight.Output.Name),
						Span:        source.Span{Start: e.Pos(), End: e.End()},
					},
				})
			} else {
				sig.Output = lca
				msgs = append(msgs, feedback.Warning{
					Classification: feedback.TypeCheckWarning,
					File:           scope.File,
					What: feedback.Selection{
						Description: fmt.Sprintf("automatically cast `%s` and `%s` to `%s`",
							sigLeft.Output.Name,
							sigRight.Output.Name,
							lca.Name),
						Span: source.Span{Start: e.Pos(), End: e.End()},
					},
				})
			}
		} else {
			sig.Output = sigLeft.Output
		}

		switch e.Operator {
		case "<":
			fallthrough
		case ">":
			fallthrough
		case "<=":
			fallthrough
		case ">=":
			fallthrough
		case "==":
			sig.Output = scope.inheritanceTree.Table["Boolean"]
		}
	case *DispatchExpr:
		if funcSig, _ := scope.lookupVariable(e.Root.Name); funcSig != nil {
			totalExpectedParams := len(funcSig.Inputs)
			totalGivenArgs := len(e.Arguments)
			sig.Output = funcSig.Output

			if totalExpectedParams == totalGivenArgs {
				for n := 0; n < totalExpectedParams; n++ {
					nthParamType := funcSig.Inputs[n]
					nthArg := e.Arguments[n]

					nthArgSignature, moreMsgs := checkExpression(scope, nthArg)
					msgs = append(msgs, moreMsgs...)

					if nthParamType != nthArgSignature.Output {
						if nthArgSignature.Output.isDescendantOf(nthParamType) {
							// TODO: possibly cast nthArgSignature.Output ->
							// nthParamType
						} else {
							msgs = append(msgs, feedback.Error{
								Classification: feedback.TypeCheckError,
								File:           scope.File,
								What: feedback.Selection{
									Description: fmt.Sprintf("%s cannot be automatically cast to type %s",
										nthArgSignature.Output.Name,
										nthParamType.Name),
									Span: source.Span{Start: nthArg.Pos(), End: nthArg.End()},
								},
							})
						}
					}
				}
			} else {
				msgs = append(msgs, feedback.Error{
					Classification: feedback.TypeCheckError,
					File:           scope.File,
					What: feedback.Selection{
						Description: fmt.Sprintf("%d arguments passed",
							totalGivenArgs),
						Span: source.Span{
							Start: e.LeftParen.Span.Start,
							End: e.RightParen.Span.End,
						},
					},
					Why: []feedback.Selection{
						{
							Description: fmt.Sprintf("`%s` expected %d arguments",
								e.Root.Name,
								totalExpectedParams),
							Span: funcSig.Definition,
						},
					},
				})
			}
		} else {
			// even though the function signature is unknown, check each
			// argument for internal type validation
			for n := 0; n < len(e.Arguments); n++ {
				nthArg := e.Arguments[n]
				_, moreMsgs := checkExpression(scope, nthArg)
				msgs = append(msgs, moreMsgs...)
			}

			// give dispatch expression a Root Type, even though lookup failed
			sig.Output = scope.inheritanceTree.RootType

			msgs = append(msgs, feedback.Error{
				Classification: feedback.TypeCheckError,
				File:           scope.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("unrecognized function `%s`", e.Root.Name),
					Span:        source.Span{Start: e.Pos(), End: e.End()},
				},
			})
		}
	case *FuncExpr:
		for _, param := range e.Parameters.Fields {
			var paramSig *Signature
			var moreMsgs []feedback.Message

			paramSig, moreMsgs = signatureFromAnnotation(scope, param)
			msgs = append(msgs, moreMsgs...)

			sig.Inputs = append(sig.Inputs, paramSig.Output)
		}

		sig.Output = scope.inheritanceTree.RootType
	default:
		fmt.Printf("UNKNOWN AST NODE: %T\n", e)
		sig.Output = scope.inheritanceTree.RootType
	}

	return sig, msgs
}

// TODO should return a Type struct that represents the proven return type of
// the function body
func checkFunctionBody(scope *Scope, sig *Signature, expr *FuncExpr) (msgs []feedback.Message) {
	subScope := scope.subScope()
	paramNames := make(map[string]bool)

	sig.Definition = source.Span{
		Start: expr.Parameters.LeftParen.Span.Start,
		End: expr.Parameters.RightParen.Span.End,
	}

	for i, param := range expr.Parameters.Fields {
		paramName := param.Identifier.Name
		paramNames[paramName] = true
		paramSig := &Signature{
			Output:     sig.Inputs[i],
			Definition: source.Span{Start: param.Pos(), End: param.End()},
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

		if t, ok = scope.inheritanceTree.Table[s.Annotation.Name]; ok == false {
			// even though the annotation references an undefined type, populate
			// the Scope's inheritance tree and type definition table with the
			// undefined type's data so that later references to the type will
			// have more useful error messages
			t = &Type{
				Name:       s.Annotation.Name,
				Parent:     scope.inheritanceTree.RootType,
				Definition: source.Span{Start: s.Annotation.Pos(), End: s.Annotation.End()},
			}

			scope.inheritanceTree.Table[s.Annotation.Name] = t

			msgs = append(msgs, feedback.Error{
				Classification: feedback.TypeCheckError,
				File:           scope.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("undefined type `%s`", s.Annotation.Name),
					Span:        t.Definition,
				},
			})
		}
	} else {
		t = scope.inheritanceTree.RootType
	}

	return &Signature{
		Output:     t,
		Definition: source.Span{Start: s.Pos(), End: s.End()},
	}, msgs
}

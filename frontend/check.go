package frontend

import (
	"fmt"

	"github.com/isaacev/Plaid/feedback"
	"github.com/isaacev/Plaid/source"
)

func Check(file *source.File, prog *ProgramNode) (msgs []feedback.Message) {
	globalScope := newGlobalScope(file)
	return checkNode(globalScope, prog)
}

func checkNode(scope *Scope, node Node) (msgs []feedback.Message) {
	switch n := node.(type) {
	case *ProgramNode:
		return checkProgramNode(scope, n)
	case Stmt:
		return checkStmt(scope, n)
	default:
		panic(fmt.Sprintf("Unknown node: %T", n))
	}
}

func checkStmt(scope *Scope, stmt Stmt) (msgs []feedback.Message) {
	switch s := stmt.(type) {
	case *IfStmt:
		return checkIfStmt(scope, s)
	case *LoopStmt:
		return checkLoopStmt(scope, s)
	case *DeclarationStmt:
		return checkDeclarationStmt(scope, s)
	case *AssignmentStmt:
		return checkAssignmentStmt(scope, s)
	case *ReturnStmt:
		return checkReturnStmt(scope, s)
	case *PrintStmt:
		return checkPrintStmt(scope, s)
	case Expr:
		return checkExpr(scope, s)
	default:
		panic(fmt.Sprintf("Unknown statement: %T", s))
	}
}

func checkExpr(scope *Scope, expr Expr) (msgs []feedback.Message) {
	switch e := expr.(type) {
	case *DispatchExpr:
		return checkDispatchExpr(scope, e)
	case *BinaryExpr:
		return checkBinaryExpr(scope, e)
	case *IdentExpr:
		return checkIdentExpr(scope, e)
	case Literal:
		return checkLiteral(scope, e)
	default:
		panic(fmt.Sprintf("Unknown expression: %T", e))
	}
}

func checkLiteral(scope *Scope, lit Literal) (msgs []feedback.Message) {
	switch l := lit.(type) {
	case *FuncLiteral:
		return checkFuncLiteral(scope, l)
	case *StrLiteral:
		return checkStrLiteral(scope, l)
	case *DecLiteral:
		return checkDecLiteral(scope, l)
	case *IntLiteral:
		return checkIntLiteral(scope, l)
	case *BoolLiteral:
		return checkBoolLiteral(scope, l)
	default:
		panic(fmt.Sprintf("Unknown literal: %T", l))
	}
}

func checkProgramNode(scope *Scope, node *ProgramNode) (msgs []feedback.Message) {
	// Determine the type for each statement at the top level of the program
	for _, stmt := range node.Statements {
		msgs = append(msgs, checkStmt(scope, stmt)...)
	}

	// Attach a list of all top-level local variables to the program node so
	// that the correct number of registers can be allocated during compilation
	for i, varName := range scope.registeredVariables {
		node.Locals = append(node.Locals, &LocalRecord{
			Name:        varName,
			LookupIndex: i,
		})
	}

	// Catch any illegal return statements outside of a function body
	for _, rec := range scope.returns {
		msgs = append(msgs, feedback.Error{
			Classification: feedback.IllegalStatementError,
			File:           scope.File,
			What: feedback.Selection{
				Description: "return statements are not allowed outside of a function body",
				Span:        rec.Span,
			},
		})
	}

	return msgs
}

func checkIfStmt(scope *Scope, stmt *IfStmt) (msgs []feedback.Message) {
	msgs = append(msgs, checkExpr(scope, stmt.Condition)...)

	if stmt.Condition.GetType().CastsTo(scope.types.builtin.Bool) == false {
		// If statement expects the test condition to evaluate to type `Bool`,
		// emit this error if the condition evaluates to some other type
		msgs = append(msgs, feedback.Error{
			Classification: feedback.MismatchedTypeError,
			File: scope.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("condition must have type `%s`, instead found `%s`",
					scope.types.builtin.Bool.String(),
					stmt.Condition.GetType().String()),
				Span: source.Span{stmt.Condition.Pos(), stmt.Condition.End()},
			},
		})
	}

	msgs = append(msgs, checkConditionalBody(scope, stmt.Body)...)
	return msgs
}

func checkLoopStmt(scope *Scope, stmt *LoopStmt) (msgs []feedback.Message) {
	msgs = append(msgs, checkExpr(scope, stmt.Condition)...)

	if stmt.Condition.GetType().CastsTo(scope.types.builtin.Bool) == false {
		// Loop statement expects the test condition to evaluate to type `Bool`,
		// emit this error if the condition evaluates to some other type
		msgs = append(msgs, feedback.Error{
			Classification: feedback.MismatchedTypeError,
			File: scope.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("condition must have type `%s`, instead found `%s`",
					scope.types.builtin.Bool.String(),
					stmt.Condition.GetType().String()),
				Span: source.Span{stmt.Condition.Pos(), stmt.Condition.End()},
			},
		})
	}

	msgs = append(msgs, checkConditionalBody(scope, stmt.Body)...)
	return msgs
}

func checkConditionalBody(scope *Scope, body *ConditionalBody) (msgs []feedback.Message) {
	for _, stmt := range body.Statements {
		msgs = append(msgs, checkStmt(scope, stmt)...)
	}

	return msgs
}

func checkDeclarationStmt(scope *Scope, stmt *DeclarationStmt) (msgs []feedback.Message) {
	if assigneeType := scope.lookupLocalVariable(stmt.Assignee.Name); assigneeType == nil {
		// Variable has not been declared yet, add its name and type top the
		// local scope
		msgs = append(msgs, checkExpr(scope, stmt.Assignment)...)

		// This is for debugging so when an error messages need to point to
		// the source code that defined a variable or a parameter's type
		// annotation, those locations are stored with the variable's type
		// in scope
		var def definition

		if funcLiteral, ok := stmt.Assignment.(*FuncLiteral); ok {
			var paramDefs []source.Span

			for _, param := range funcLiteral.Parameters {
				paramDefs = append(paramDefs, source.Span{
					param.Name.Pos(),
					param.Name.End(),
				})
			}

			var returnDef source.Span

			if funcLiteral.ReturnAnnotation != nil {
				returnDef = source.Span{
					funcLiteral.ReturnAnnotation.Pos(),
					funcLiteral.ReturnAnnotation.End(),
				}
			}

			def = definition{
				wholeDef:  source.Span{stmt.Pos(), stmt.End()},
				paramDefs: paramDefs,
				returnDef: returnDef,
			}
		} else {
			def = definition{
				wholeDef: source.Span{stmt.Pos(), stmt.End()},
			}
		}

		scope.registerLocalVariable(stmt.Assignee.Name, stmt.Assignment.GetType(), def)

		// In order to support recursion it's necessary that a function's type be
		// bound to the variable in the enclosing scope BEFORE the function's body
		// is analyzed so that any recursive calls will be treated as references to
		// an upvalue outside the function closure
		if funcLiteral, ok := stmt.Assignment.(*FuncLiteral); ok {
			msgs = append(msgs, checkFuncBody(scope, funcLiteral)...)
		}
	} else {
		// If the identifier used in a declaration statement has already been
		// defined LOCALLY, it counts as an illegal re-declaration of the
		// variable
		msg := feedback.Error{
			Classification: feedback.RedeclarationError,
			File:           scope.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("`%s` redeclared here",
					stmt.Assignee.Name),
				Span: source.Span{stmt.Assignee.Pos(), stmt.Assignee.End()},
			},
		}

		if exists, def := scope.lookupLocalVariableDefinition(stmt.Assignee.Name); exists {
			msg.Why = append(msg.Why, feedback.Selection{
				Description: fmt.Sprintf("`%s` originally declared here", stmt.Assignee.Name),
				Span:        def.wholeDef,
			})
		}

		msgs = append(msgs, msg)
	}

	return msgs
}

func checkAssignmentStmt(scope *Scope, stmt *AssignmentStmt) (msgs []feedback.Message) {
	if assigneeType, _ := scope.lookupVariable(stmt.Assignee.Name); assigneeType == nil {
		// Variable is being assigned before it has been declared
		msgs = append(msgs, feedback.Error{
			Classification: feedback.UndefinedVariableError,
			File:           scope.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("variable `%s` cannot be assigned before it has been declared",
					stmt.Assignee.Name),
				Span: source.Span{stmt.Pos(), stmt.End()},
			},
		})
	}

	// Check the type of the expression being assigned to the assignee
	checkExpr(scope, stmt.Assignment)

	// In order to support recursion it's necessary that a function's type be
	// bound to the variable in the enclosing scope BEFORE the function's body
	// is analyzed so that any recursive calls will be treated as references to
	// an upvalue outside the function closure
	if funcLiteral, ok := stmt.Assignment.(*FuncLiteral); ok {
		msgs = append(msgs, checkFuncBody(scope, funcLiteral)...)
	}

	return msgs
}

func checkReturnStmt(scope *Scope, stmt *ReturnStmt) (msgs []feedback.Message) {
	record := ReturnRecord{
		Type: nil,
		Span: source.Span{stmt.Pos(), stmt.End()},
	}

	if stmt.Argument != nil {
		msgs = checkExpr(scope, stmt.Argument)
		record.Type = stmt.Argument.GetType()

		// Check the bodies of any anonymous functions being returned
		if funcLiteral, ok := stmt.Argument.(*FuncLiteral); ok {
			msgs = append(msgs, checkFuncBody(scope, funcLiteral)...)
		}
	}

	scope.returns = append(scope.returns, record)
	return msgs
}

func checkPrintStmt(scope *Scope, stmt *PrintStmt) (msgs []feedback.Message) {
	// As long as the print arguments are expressions, don't worry about
	// their result types since the statement handles any type internally.
	// Just ensure that each expression is internally valid
	for _, arg := range stmt.Arguments {
		msgs = append(msgs, checkExpr(scope, arg)...)

		// Check the bodies of any anonymous functions being passed as an
		// argument to the print statement
		if funcLiteral, ok := arg.(*FuncLiteral); ok {
			msgs = append(msgs, checkFuncBody(scope, funcLiteral)...)
		}
	}

	return msgs
}

func checkDispatchExpr(scope *Scope, expr *DispatchExpr) (msgs []feedback.Message) {
	// By default set the expression's type to `Any`. This will likely be
	// overwritten but prevents it from becoming `nil`
	expr.SetType(scope.types.builtin.Any)

	msgs = append(msgs, checkExpr(scope, expr.Root)...)
	calleeType := expr.Root.GetType()

	if calleeType.Equals(scope.types.builtin.Any) {
		// Since the callee has type `Any`, no type information can be
		// determined about what types the arguments should have. Nonetheless,
		// ensure that each argument expression is internally consistent
		for _, arg := range expr.Arguments {
			msgs = append(msgs, checkExpr(scope, arg)...)

			// Check the bodies of any anonymous functions being passed as an
			// argument to the function
			if funcLiteral, ok := arg.(*FuncLiteral); ok {
				msgs = append(msgs, checkFuncBody(scope, funcLiteral)...)
			}
		}
	} else if calleeFuncType, ok := calleeType.(*FuncType); ok == false {
		msgs = append(msgs, feedback.Error{
			Classification: feedback.IllegalFunctionCall,
			File:           scope.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("Cannot use expression with type `%s` as a function",
					calleeType.String()),
				Span: source.Span{expr.Root.Pos(), expr.Root.End()},
			},
		})
	} else {
		for _, arg := range expr.Arguments {
			msgs = append(msgs, checkExpr(scope, arg)...)
		}

		totalExpectedArgs := len(calleeFuncType.params)
		totalGivenArgs := len(expr.Arguments)

		if totalExpectedArgs == totalGivenArgs {
			for n, nthArg := range expr.Arguments {
				nthArgType := nthArg.GetType()
				nthParamType := calleeFuncType.params[n]

				if nthArgType.CastsTo(nthParamType) == false {
					// Mismatched argument type
					msgs = append(msgs, feedback.Error{
						Classification: feedback.MismatchedTypeError,
						File:           scope.File,
						What: feedback.Selection{
							Description: fmt.Sprintf("`%s` cannot be used as type `%s`",
								nthArgType.String(),
								nthParamType.String()),
							Span: source.Span{nthArg.Pos(), nthArg.End()},
						},
					})

					// IDEA: when an argument is passed with the wrong type, add
					// a `Why` clause which points to the parameter definition
					// in the function signature with the description:
					// "type `Int` expected as 2nd argument"
				}
			}

			expr.SetType(calleeFuncType.returnType)
		} else {
			// Wrong number of arguments given during function call
			msgs = append(msgs, feedback.Error{
				Classification: feedback.MismatchedArgumentsError,
				File:           scope.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("Expected %d arguments, received %d",
						totalExpectedArgs,
						totalGivenArgs),
					Span: source.Span{expr.LeftParen.Span.Start, expr.RightParen.Span.End},
				},
			})
		}
	}

	return msgs
}

func checkBinaryExpr(scope *Scope, expr *BinaryExpr) (msgs []feedback.Message) {
	// By default set the expression's type to `Any`. This will likely be
	// overwritten but prevents it from becoming `nil`
	expr.SetType(scope.types.builtin.Any)

	// Store the operator string in a variable for easy access
	op := string(expr.Operator)

	// Type check the left operand
	msgs = append(msgs, checkExpr(scope, expr.Left)...)
	typeLeft := expr.Left.GetType()

	// Type check the right operand
	msgs = append(msgs, checkExpr(scope, expr.Right)...)
	typeRight := expr.Right.GetType()

	if typeLeft.Equals(scope.types.builtin.Any) || typeRight.Equals(scope.types.builtin.Any) {
		// One or both of the operands are type `Any` so no operand type-checks
		// are performed
		if exists, resultType := scope.types.builtin.Any.HasMethod(op, scope.types.builtin.Any); exists {
			// While the type `Any` can be used with any binary operation, there
			// are some operations (mostly comparison operations) which still
			// have a known output type (mostly `Bool`) so it's necessary to
			// check if `Any` has a defined method for the current binary operation
			expr.SetType(resultType)
		}
	} else {
		// Both the left and right operands are statically typed, nether is `Any`
		if exists, resultType := typeLeft.HasMethod(op, typeRight); exists {
			expr.SetType(resultType)
		} else {
			// The left operand's type has no method defined for the given
			// binary operation and the given right operand's type
			msgs = append(msgs, feedback.Error{
				Classification: feedback.MismatchedTypeError,
				File: scope.File,
				What: feedback.Selection{
					Description: fmt.Sprintf("type `%s` cannot call `%s` with type `%s`",
						typeLeft.String(),
						op,
						typeRight.String()),
					Span: source.Span{expr.Pos(), expr.End()},
				},
			})

			expr.SetType(typeLeft)
		}
	}

	return msgs
}

func checkIdentExpr(scope *Scope, expr *IdentExpr) (msgs []feedback.Message) {
	// By default set the expression's type to `Any`. This will likely be
	// overwritten but prevents it from becoming `nil`
	expr.SetType(scope.types.builtin.Any)

	if t, isLocal := scope.lookupVariable(expr.Name); t == nil {
		// Emit this error if an identifier is used whose corresponding type
		// can't be found in the local scope, any enclosing scopes or the global
		// scope
		msgs = append(msgs, feedback.Error{
			Classification: feedback.UndefinedVariableError,
			File:           scope.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("variable `%s` is undeclared",
					expr.Name),
				Span: source.Span{expr.Pos(), expr.End()},
			},
		})
	} else if isLocal {
		// Variable is local and already declared, so tag it with the
		// appropriate type registered in the type table
		expr.SetType(t)
	} else {
		// Register identifier as an upvalue since it was defined in an
		// enclosing scope
		scope.registerUpvalue(expr.Name)
		expr.SetType(t)
	}

	return msgs
}

func checkFuncLiteral(scope *Scope, expr *FuncLiteral) (msgs []feedback.Message) {
	var paramTypes []Type
	var returnType Type
	var err feedback.Message

	// Convert the type annotation of each function parameter to a type
	// signature. If no annotation is given, the parameter is automatically
	// given the `Any` type by the `typeAnnotationToType` function
	for _, param := range expr.Parameters {
		var paramType Type

		if paramType, err = typeAnnotationToType(scope, param.Annotation); err != nil {
			msgs = append(msgs, err)
		}

		paramTypes = append(paramTypes, paramType)
	}

	// Convert the return type annotation to a type signature
	if returnType, err = typeAnnotationToType(scope, expr.ReturnAnnotation); err != nil {
		msgs = append(msgs, err)
	}

	// Create a new type tag for the function literal
	expr.SetType(&FuncType{
		params:     paramTypes,
		returnType: returnType,
	})

	return msgs
}

func checkFuncBody(scope *Scope, expr *FuncLiteral) (msgs []feedback.Message) {
	// Create a lexical scope for the function
	funcScope := scope.subScope()

	// Add an easy lookup map so that a given local variable can be quickly
	// classified as a parameter or not
	paramNames := make(map[string]bool)

	// Add function parameters and types to function scope
	for i, param := range expr.Parameters {
		funcScope.registerLocalVariable(param.Name.Name, expr._type.params[i], definition{
			wholeDef: source.Span{param.Pos(), param.End()},
		})
		paramNames[param.Name.Name] = true
	}

	// Check each statement inside the function body in the context of the
	// function scope
	for _, stmt := range expr.Body.Statements {
		msgs = append(msgs, checkStmt(funcScope, stmt)...)
	}

	// After the statements have been recursively checked, the scope has
	// accumulated a list of all `return` statements in the function. Check each
	// statement's return type to make sure it matches the function's return
	// type and emit an error if it doesn't. If the function has the return type
	// of `Any`, don't do these checks because the function can return any type
	if expr._type.returnType.Equals(scope.types.builtin.Any) == false {
		for _, returnRecord := range funcScope.returns {
			if returnRecord.Type.CastsTo(expr._type.returnType) == false {
				msgs = append(msgs, feedback.Error{
					Classification: feedback.MismatchedTypeError,
					File: funcScope.File,
					What: feedback.Selection{
						Description: fmt.Sprintf("... but function tried to return type `%s`",
							returnRecord.Type.String()),
						Span: returnRecord.Span,
					},
					Why: []feedback.Selection{
						{
							Description: fmt.Sprintf("function expects return type `%s`...",
								expr._type.returnType.String()),
							Span: source.Span{expr.ReturnAnnotation.Pos(), expr.ReturnAnnotation.End()},
						},
					},
				})
			}
		}
	}

	// Include a list of all local variables with some metadata for use during
	// register allocation
	for i, varName := range funcScope.registeredVariables {
		expr.Locals = append(expr.Locals, &LocalRecord{
			Name:        varName,
			IsParameter: paramNames[varName],
			LookupIndex: i,
		})
	}

	// Include a list of the function scope's upvalues (references to variables
	// declared up the scope chain) so that during compilation this references
	// can be built into the bytecode
	for _, record := range funcScope.upvalues {
		expr.Upvalues = append(expr.Upvalues, record)
	}

	return msgs
}

func checkStrLiteral(scope *Scope, expr *StrLiteral) (msgs []feedback.Message) {
	expr.SetType(scope.types.builtin.Str)
	return msgs
}

func checkDecLiteral(scope *Scope, expr *DecLiteral) (msgs []feedback.Message) {
	expr.SetType(scope.types.builtin.Dec)
	return msgs
}

func checkIntLiteral(scope *Scope, expr *IntLiteral) (msgs []feedback.Message) {
	expr.SetType(scope.types.builtin.Int)
	return msgs
}

func checkBoolLiteral(scope *Scope, expr *BoolLiteral) (msgs []feedback.Message) {
	expr.SetType(scope.types.builtin.Bool)
	return msgs
}

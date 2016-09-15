package frontend

import (
	"fmt"
	"strings"
)

func StringifyAST(prog *ProgramNode, showTypes bool) string {
	return stringifyNode(prog, showTypes)
}

func stringifyNode(generic Node, showTypes bool) string {
	const newline = "\n"

	switch node := generic.(type) {
	case *ProgramNode:
		block := ""

		for i := 0; i < len(node.Statements); i++ {
			block += stringifyNode(node.Statements[i], showTypes)

			if i+1 < len(node.Statements) {
				block += newline
			}
		}

		return fmt.Sprintf("(program (locals=%d upvalues=%d) (\n%s\n))",
			len(node.Locals),
			len(node.Upvalues),
			indentString(block))
	case *FuncLiteral:
		if showTypes {
			return fmt.Sprintf("(func (locals=%d upvalues=%v) %s: %s %s)",
				len(node.Locals),
				len(node.Upvalues),
				stringifyParams(node.Parameters, showTypes),
				node._type.returnType.String(),
				stringifyNode(node.Body, showTypes))
		} else {
			return fmt.Sprintf("(func (locals=%d upvalues=%v) %s: %s)",
				len(node.Locals),
				len(node.Upvalues),
				stringifyParams(node.Parameters, showTypes),
				stringifyNode(node.Body, showTypes))
		}
	case *FuncBody:
		var body string

		for i, stmt := range node.Statements {
			body += stringifyNode(stmt, showTypes)

			if i < len(node.Statements)-1 {
				body += newline
			}
		}

		return fmt.Sprintf("(\n%s\n)",
			indentString(body))
	case *IfStmt:
		// return fmt.Sprintf("(if %s %s)",
		// 	stringifyNode(node.IfClause.Condition),
		// 	stringifyNode(node.IfClause.Body))

		str := fmt.Sprintf("(if %s %s",
			stringifyNode(node.IfClause.Condition, showTypes),
			stringifyNode(node.IfClause.Body, showTypes))

		for _, clause := range node.ElifClauses {
			str += fmt.Sprintf(" elif %s %s",
				stringifyNode(clause.Condition, showTypes),
				stringifyNode(clause.Body, showTypes))
		}

		if node.ElseClause != nil {
			str += fmt.Sprintf(" else %s",
				stringifyNode(node.ElseClause.Body, showTypes))
		}

		str += ")"

		return str
	case *LoopStmt:
		return fmt.Sprintf("(loop %s %s)",
			stringifyNode(node.Clause.Condition, showTypes),
			stringifyNode(node.Clause.Body, showTypes))
	case *ClauseBody:
		var body string

		for i, stmt := range node.Statements {
			body += stringifyNode(stmt, showTypes)

			if i < len(node.Statements)-1 {
				body += "\n"
			}
		}

		return fmt.Sprintf("(\n%s\n)",
			indentString(body))
	case *PrintStmt:
		// TODO improve to handle 0 or 2+ arguments
		return fmt.Sprintf("(print %s)",
			stringifyNode(node.Arguments[0], showTypes))
	case *ReturnStmt:
		if node.Argument != nil {
			return fmt.Sprintf("(return %s)",
				stringifyNode(node.Argument, showTypes))
		} else {
			return "(return)"
		}
	case *DeclarationStmt:
		return fmt.Sprintf("(let \"%s\" %s)",
			node.Assignee.Name,
			stringifyNode(node.Assignment, showTypes))
	case *AssignmentStmt:
		return fmt.Sprintf("(set \"%s\" %s)",
			node.Assignee.Name,
			stringifyNode(node.Assignment, showTypes))
	case *DispatchExpr:
		var args string

		for i, arg := range node.Arguments {
			args += stringifyNode(arg, showTypes)

			if i < len(node.Arguments)-1 {
				args += " "
			}
		}

		if showTypes {
			return fmt.Sprintf("[%s (%s %s)]",
				stringifyType(node),
				stringifyNode(node.Root, showTypes),
				args)
		} else {
			return fmt.Sprintf("(%s %s)",
				stringifyNode(node.Root, showTypes),
				args)
		}
	case *MethodDispatchExpr:
		var args string

		for i, arg := range node.Arguments {
			args += stringifyNode(arg, showTypes)

			if i < len(node.Arguments)-1 {
				args += " "
			}
		}

		if args == "" {
			args = "<no args>"
		}

		if showTypes {
			return fmt.Sprintf("[%s (%s.%s %s)]",
				stringifyType(node),
				stringifyNode(node.Root, showTypes),
				stringifyNode(node.Method, showTypes),
				args)
		} else {
			return fmt.Sprintf("(%s.%s %s)",
				stringifyNode(node.Root, showTypes),
				stringifyNode(node.Method, showTypes),
				args)
		}
	case *IndexAccessExpr:
		if showTypes {
			return fmt.Sprintf("[%s %s at %s]",
				stringifyType(node),
				stringifyNode(node.Root, showTypes),
				stringifyNode(node.Index, showTypes))
		} else {
			return fmt.Sprintf("%s at %s",
				stringifyNode(node.Root, showTypes),
				stringifyNode(node.Index, showTypes))
		}
	case *UnaryExpr:
		// NOTE: the AST-stringified representation of unary expressions ignores
		// the prefix/postfix distinction since that is largely a syntactic
		// issue and does not bear heavily on the underlying structure of the
		// syntax tree
		if showTypes {
			return fmt.Sprintf("[%s (%s %s)]",
				stringifyType(node),
				string(node.Operator.Symbol),
				stringifyNode(node.Operand, showTypes))
		} else {
			return fmt.Sprintf("(%s %s)",
				string(node.Operator.Symbol),
				stringifyNode(node.Operand, showTypes))
		}
	case *BinaryExpr:
		if showTypes {
			return fmt.Sprintf("[%s (%s %s %s)]",
				stringifyType(node),
				string(node.Operator),
				stringifyNode(node.Left, showTypes),
				stringifyNode(node.Right, showTypes))
		} else {
			return fmt.Sprintf("(%s %s %s)",
				string(node.Operator),
				stringifyNode(node.Left, showTypes),
				stringifyNode(node.Right, showTypes))
		}
	case *IdentExpr:
		if showTypes {
			return fmt.Sprintf("[%s %s]",
				stringifyType(node),
				node.Name)
		} else {
			return node.Name
		}
	case *ListLiteral:
		var elements string

		for i, elem := range node.Elements {
			elements += stringifyNode(elem, showTypes)

			if i < len(node.Elements)-1 {
				elements += ", "
			}
		}

		if len(node.Elements) == 0 {
			elements = "<empty>"
		}

		if showTypes {
			return fmt.Sprintf("[%s (list %s)]",
				stringifyType(node),
				elements)
		} else {
			return fmt.Sprintf("(list %s)",
				elements)
		}
	case *IntLiteral:
		if showTypes {
			return fmt.Sprintf("[%s %d]",
				stringifyType(node),
				node.Value)
		} else {
			return fmt.Sprintf("%d",
				node.Value)
		}
	case *DecLiteral:
		if showTypes {
			return fmt.Sprintf("[%s %.2f]",
				stringifyType(node),
				node.Value)
		} else {
			return fmt.Sprintf("%.2f",
				node.Value)
		}
	case *TemplateLiteral:
		var contents string

		for i, str := range node.Strings {
			contents += stringifyNode(str, showTypes)

			if i < len(node.Expressions) {
				contents += ", " + stringifyNode(node.Expressions[i], showTypes)
			}
		}

		return fmt.Sprintf("(%s)",
			contents)
	case *StrLiteral:
		return fmt.Sprintf("`%s`",
			node.Value)
	case *BoolLiteral:
		return fmt.Sprintf("<%t>",
			node.Value)
	default:
		return fmt.Sprintf("<Unknown %T>", node)
	}
}

func indentString(s string) string {
	lines := strings.Split(s, "\n")

	for i, l := range lines {
		lines[i] = "   " + l
	}

	return strings.Join(lines, "\n")
}

func stringifyType(expr Expr) string {
	t := expr.GetType()

	if t == nil {
		return "???"
	}

	switch t := expr.GetType().(type) {
	case *TypeOperator:
		if t == nil {
			return "???"
		}
	case *FuncType:
		if t == nil {
			return "???"
		}
	}

	return t.String()
}

func stringifyParams(params []*Parameter, showTypes bool) string {
	out := "("

	for i, param := range params {
		if param.Annotation == nil {
			out += param.Name.Name + ": Any"
		} else {
			if showTypes {
				out += fmt.Sprintf("%s: %s",
					param.Name.Name,
					stringifyAnnotation(param.Annotation))
			} else {
				out += param.Name.Name
			}
		}

		if i < len(params)-1 {
			out += ", "
		}
	}

	out += ")"
	return out
}

func stringifyAnnotation(ta TypeAnnotation) string {
	switch t := ta.(type) {
	case NamedTypeAnnotation:
		return t.Name.Name
	case FuncTypeAnnotation:
		out := stringifyAnnotations(t.Parameters...) + " => "

		if _, ok := t.ReturnType.(FuncTypeAnnotation); ok {
			out += fmt.Sprintf("(%s)",
				stringifyAnnotation(t.ReturnType))
		} else {
			out += stringifyAnnotation(t.ReturnType)
		}

		return out
	case ListTypeAnnotation:
		return fmt.Sprintf("[%s]", stringifyAnnotation(t.ElementType))
	default:
		return "Any"
	}
}

func stringifyAnnotations(tas ...TypeAnnotation) string {
	if len(tas) == 1 {
		return stringifyAnnotation(tas[0])
	}

	out := "("

	for i, ta := range tas {
		out += stringifyAnnotation(ta)

		if i < len(tas)-1 {
			out += ", "
		}
	}

	out += ")"
	return out
}

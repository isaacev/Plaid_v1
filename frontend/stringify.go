package frontend

import (
	"fmt"
	"strings"
)

func StringifyAST(prog *ProgramNode) string {
	return stringifyNode(prog)
}

func stringifyNode(generic Node) string {
	const newline = "\n"

	switch node := generic.(type) {
	case *ProgramNode:
		block := ""

		for i := 0; i < len(node.Statements); i++ {
			block += stringifyNode(node.Statements[i])

			if i+1 < len(node.Statements) {
				block += newline
			}
		}

		return fmt.Sprintf("(program (locals=%d upvalues=%d) (\n%s\n))",
			len(node.Locals),
			len(node.Upvalues),
			indentString(block))
	case *FuncLiteral:
		return fmt.Sprintf("(func (locals=%d upvalues=%v) %s: %s %s)",
			len(node.Locals),
			len(node.Upvalues),
			stringifyParams(node.Parameters),
			node._type.returnType.String(),
			stringifyNode(node.Body))
	case *FuncBody:
		var body string

		for i, stmt := range node.Statements {
			body += stringifyNode(stmt)

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
			stringifyNode(node.IfClause.Condition),
			stringifyNode(node.IfClause.Body))

		for _, clause := range node.ElifClauses {
			str += fmt.Sprintf(" elif %s %s",
				stringifyNode(clause.Condition),
				stringifyNode(clause.Body))
		}

		if node.ElseClause != nil {
			str += fmt.Sprintf(" else %s",
				stringifyNode(node.ElseClause.Body))
		}

		str += ")"

		return str
	case *ClauseBody:
		var body string

		for i, stmt := range node.Statements {
			body += stringifyNode(stmt)

			if i < len(node.Statements)-1 {
				body += "\n"
			}
		}

		return fmt.Sprintf("(\n%s\n)",
			indentString(body))
	case *PrintStmt:
		// TODO improve to handle 0 or 2+ arguments
		return fmt.Sprintf("(print %s)",
			stringifyNode(node.Arguments[0]))
	case *ReturnStmt:
		if node.Argument != nil {
			return fmt.Sprintf("(return %s)",
				stringifyNode(node.Argument))
		} else {
			return "(return)"
		}
	case *DeclarationStmt:
		return fmt.Sprintf("(let \"%s\" %s)",
			node.Assignee.Name,
			stringifyNode(node.Assignment))
	case *AssignmentStmt:
		return fmt.Sprintf("(set \"%s\" %s)",
			stringifyNode(node.Assignee),
			stringifyNode(node.Assignment))
	case *DispatchExpr:
		var args string

		for i, arg := range node.Arguments {
			args += stringifyNode(arg)

			if i < len(node.Arguments)-1 {
				args += " "
			}
		}

		return fmt.Sprintf("[%s (%s %s)]",
			stringifyType(node),
			stringifyNode(node.Root),
			args)
	case *IndexAccessExpr:
		return fmt.Sprintf("[%s %s at %s]",
			stringifyType(node),
			stringifyNode(node.Root),
			stringifyNode(node.Index))
	case *BinaryExpr:
		return fmt.Sprintf("[%s (%s %s %s)]",
			stringifyType(node),
			string(node.Operator),
			stringifyNode(node.Left),
			stringifyNode(node.Right))
	case *IdentExpr:
		return fmt.Sprintf("[%s %s]",
			stringifyType(node),
			node.Name)
	case *ListLiteral:
		var elements string

		for i, elem := range node.Elements {
			elements += stringifyNode(elem)

			if i < len(node.Elements)-1 {
				elements += ", "
			}
		}

		if len(node.Elements) == 0 {
			elements = "<empty>"
		}

		return fmt.Sprintf("[%s (list %s)]", stringifyType(node), elements)
	case *IntLiteral:
		return fmt.Sprintf("[%s %d]", stringifyType(node), node.Value)
	case *DecLiteral:
		return fmt.Sprintf("[%s %.2f]", stringifyType(node), node.Value)
	case *StrLiteral:
		return fmt.Sprintf("[%s `%s`]", stringifyType(node), node.Value)
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

func stringifyParams(params []*Parameter) string {
	out := "("

	for i, param := range params {
		if param.Annotation == nil {
			out += param.Name.Name + ": Any"
		} else {
			out += fmt.Sprintf("%s: %s",
				param.Name.Name,
				stringifyAnnotation(param.Annotation))
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

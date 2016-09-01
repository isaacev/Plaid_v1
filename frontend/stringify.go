package frontend

import (
	"fmt"
	"strings"
)

func StringifyAST(prog *Program) string {
	return stringifyNode(prog)
}

func stringifyNode(generic Node) string {
	const newline = "\n"

	switch node := generic.(type) {
	case *Program:
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
	case *FuncExpr:
		return fmt.Sprintf("(func (locals=%d upvalues=%v) %s %s)",
			len(node.Locals),
			len(node.Upvalues),
			stringifyNode(node.Parameters),
			stringifyNode(node.Body))
	case *FieldList:
		fields := "("

		for i, field := range node.Fields {
			fields += stringifyNode(field)

			if i < len(node.Fields)-1 {
				fields += ", "
			}
		}

		return fields + ")"
	case *TypeAnnotationStmt:
		var typeStr string

		if node.ExplicitType {
			typeStr = node.Annotation.Name
		} else {
			typeStr = "Any"
		}

		return fmt.Sprintf("[%s %s]", typeStr, node.Identifier.Name)
	case *FunctionBody:
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
		return fmt.Sprintf("(if %s %s)",
			stringifyNode(node.Condition),
			stringifyNode(node.Body))
	case *ConditionalBody:
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
		if len(node.Arguments) > 0 {
			var args string

			for i, arg := range node.Arguments {
				args += stringifyNode(arg)

				if i < len(node.Arguments)-1 {
					args += ", "
				}
			}

			return fmt.Sprintf("(return %s)", args)
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

		return fmt.Sprintf("(\"%s\" %s)",
			node.Root.Name,
			args)
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
	case *IntegerExpr:
		return fmt.Sprintf("[%s %d]", stringifyType(node), node.Value)
	case *DecimalExpr:
		return fmt.Sprintf("[%s %.2f]", stringifyType(node), node.Value)
	case *StringExpr:
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
	t := expr.Type()

	if t == nil {
		return "<nil>"
	}

	return t.Name
}

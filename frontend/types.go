package frontend

import (
	"fmt"
	"strings"

	"github.com/isaacev/Plaid/feedback"
	"github.com/isaacev/Plaid/source"
)

type Type interface {
	Equals(Type) bool
	CastsTo(Type) bool
	AddMethod(*Method)
	HasMethod(string, Type) (bool, Type)
	String() string
	isType()
}

type Method struct {
	operator string
	root     Type
	operand  Type
	result   Type
}

type AnyType struct {
	methods []*Method
}

func (*AnyType) Equals(t2 Type) bool {
	if _, ok := t2.(*AnyType); ok {
		return true
	}

	return false
}

func (any *AnyType) CastsTo(t2 Type) bool {
	// It's important to remember that all types can be used where `Any` is
	// accepted but `Any` can only be used where `Any` is accepted so `Any` can
	// only be automatically cast to `Any` hence the check for equality in this
	// method
	return any.Equals(t2)
}

func (any *AnyType) AddMethod(method *Method) {
	any.methods = append(any.methods, method)
}

func (any *AnyType) HasMethod(operator string, operand Type) (exists bool, returnType Type) {
	for _, method := range any.methods {
		if method.operator == operator && method.operand.Equals(operand) {
			return true, method.result
		}
	}

	return false, nil
}

func (*AnyType) String() string {
	return "Any"
}

func (AnyType) isType() {}

type TypeOperator struct {
	name    string
	types   []Type
	methods []*Method
}

func (op *TypeOperator) Equals(t2 Type) bool {
	switch v := t2.(type) {
	case *TypeOperator:
		// Check if the two types are pointers to the same address
		if op == v {
			return true
		}

		// Check if the operator names are the same
		if op.name != v.name {
			return false
		}

		// Check that their Type arguments are the same
		if len(op.types) == len(v.types) {
			for i, t := range op.types {
				if t.Equals(v.types[i]) == false {
					return false
				}
			}

			return true
		}
	}

	return false
}

func (op *TypeOperator) CastsTo(t2 Type) bool {
	switch v := t2.(type) {
	case *AnyType:
		return true
	case *TypeOperator:
		// Check if the two types are pointers to the same address
		if op == v {
			return true
		}

		// Check if the operator names are the same
		if op.name != v.name {
			return false
		}

		// Check that their Type arguments are the same
		if len(op.types) == len(v.types) {
			for i, t := range op.types {
				if t.CastsTo(v.types[i]) == false {
					return false
				}
			}

			return true
		}
	}

	return false
}

func (op *TypeOperator) AddMethod(method *Method) {
	op.methods = append(op.methods, method)
}

func (op TypeOperator) HasMethod(operator string, operand Type) (exists bool, returnType Type) {
	for _, method := range op.methods {
		if method.operator == operator && method.operand.Equals(operand) {
			return true, method.result
		}
	}

	return false, nil
}

func (op TypeOperator) String() string {
	switch len(op.types) {
	case 0:
		return op.name
	case 2:
		return fmt.Sprintf("(%s %s %s)",
			op.types[0],
			op.name,
			op.types[1])
	default:
		strungTypes := make([]string, len(op.types))

		for i, t := range op.types {
			strungTypes[i] = t.String()
		}

		return fmt.Sprintf("%s %s",
			op.name,
			strings.Join(strungTypes, ", "))
	}
}

func (TypeOperator) isType() {}

type FuncType struct {
	params     []Type
	returnType Type
	methods    []*Method
}

func (fn *FuncType) Equals(t2 Type) bool {
	switch v := t2.(type) {
	case *FuncType:
		// Check if the two types are pointers to the same address
		if fn == v {
			return true
		}

		// Check that the return type is the same
		if fn.returnType.Equals(v.returnType) == false {
			return false
		}

		// Check that the arguments have the same types
		if len(fn.params) == len(v.params) {
			for i, t := range fn.params {
				if t.Equals(v.params[i]) == false {
					return false
				}
			}

			return true
		} else {
			return false
		}
	default:
		return false
	}
}

func (fn *FuncType) CastsTo(t2 Type) bool {
	switch v := t2.(type) {
	case *AnyType:
		return true
	case *FuncType:
		// Check if the two types are pointers to the same address
		if fn == v {
			return true
		}

		// Check that the return type is the same
		if fn.returnType.Equals(v.returnType) == false {
			return false
		}

		// Check that the arguments have the same types
		if len(fn.params) == len(v.params) {
			for i, t := range fn.params {
				if t.CastsTo(v.params[i]) == false {
					return false
				}
			}

			return true
		} else {
			return false
		}
	default:
		return false
	}
}

func (fn *FuncType) AddMethod(method *Method) {
	fn.methods = append(fn.methods, method)
}

func (fn FuncType) HasMethod(operator string, operand Type) (exists bool, returnType Type) {
	for _, method := range fn.methods {
		if method.operator == operator && method.operand.Equals(operand) {
			return true, method.result
		}
	}

	return false, nil
}

func (fn FuncType) String() string {
	if len(fn.params) == 1 {
		return fmt.Sprintf("(%s => %s)",
			fn.params[0].String(),
			fn.returnType.String())
	}

	return fmt.Sprintf("(%s => %s)",
		tupleToString(fn.params),
		fn.returnType.String())
}

func (FuncType) isType() {}

func tupleToString(tuple []Type) string {
	str := "("

	for i, t := range tuple {
		str += t.String()

		if i < len(tuple)-1 {
			str += ", "
		}
	}

	str += ")"
	return str
}

func typeAnnotationToType(scope *Scope, annotation TypeAnnotation) (Type, feedback.Message) {
	switch a := annotation.(type) {
	case NamedTypeAnnotation:
		name := a.Name.Name
		newRef := &TypeOperator{
			name: a.Name.Name,
		}

		if exists, typeRef := scope.types.getNamedType(name); exists && typeRef.CastsTo(newRef) {
			return typeRef, nil
		}

		return scope.types.builtin.Any, feedback.Error{
			Classification: feedback.UndefinedTypeError,
			File:           scope.File,
			What: feedback.Selection{
				Description: fmt.Sprintf("Unknown type `%s`", name),
				Span:        source.Span{a.Pos(), a.End()},
			},
		}
	case FuncTypeAnnotation:
		var params []Type

		for _, p := range a.Parameters {
			if paramType, err := typeAnnotationToType(scope, p); err != nil {
				return scope.types.builtin.Any, err
			} else {
				params = append(params, paramType)
			}
		}

		if returnType, err := typeAnnotationToType(scope, a.ReturnType); err != nil {
			return scope.types.builtin.Any, err
		} else {
			return &FuncType{
				params:     params,
				returnType: returnType,
			}, nil
		}
	case nil:
		return scope.types.builtin.Any, nil
	default:
		panic(fmt.Sprintf("Unknown annotation: %T", annotation))
	}
}

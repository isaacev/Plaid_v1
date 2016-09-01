package frontend

import (
	"github.com/isaacev/Plaid/source"
)

// Type represents a type defined in a program. Types can be tagged with the
// line/column location of its definition
type Type struct {
	Name       string
	Methods    []*Method
	Definition source.Span
}

func (t *Type) addMethod(method *Method) {
	t.Methods = append(t.Methods, method)
}

func (t *Type) hasMethod(operator string, operand *Type) (bool, *Type) {
	for _, method := range t.Methods {
		if method.Operator == operator && method.Operand == operand {
			return true, method.Result
		}
	}

	return false, nil
}

type Method struct {
	Operator string
	Root     *Type
	Operand  *Type
	Result   *Type
}

// Signature includes fields for an expression's input types and a single output
// type. Signatures can also be tagged with the line/column information for
// where it was defined
type Signature struct {
	Output           *Type
	Inputs           []*Type
	InputDefinitions []source.Span
	Definition       source.Span
}

// typeTable represents the relationships between Types. The table
// structure is used to determine how disparate types can be coerced into a
// single shared ancestor. All types can be coerced to the Root Type
type typeTable struct {
	AnyType *Type
	Table   map[string]*Type
}

// addType adds a new Type to the given type table. If the type is the
// first to be added to the table, it becomes the Root Type. All types are stored
// in a map of type names -> *Type structs. The map provides a system for
// arbitrary type node lookup by type name
func (table *typeTable) addType(t *Type) {
	if table.AnyType == nil {
		table.AnyType = t
	}

	table.Table[t.Name] = t
}

// newTypeTable builds a new table structure and populates it with built-in
// types including the relationships between those built-in types
func newTypeTable() *typeTable {
	table := &typeTable{
		AnyType: nil,
		Table:   make(map[string]*Type),
	}

	a := &Type{Name: "Any"}
	b := &Type{Name: "Bool"}
	i := &Type{Name: "Int"}
	d := &Type{Name: "Dec"}
	s := &Type{Name: "Str"}

	// Any logical methods
	a.addMethod(&Method{Operator: "<", Root: a, Operand: a, Result: b})
	a.addMethod(&Method{Operator: "<=", Root: a, Operand: a, Result: b})
	a.addMethod(&Method{Operator: ">", Root: a, Operand: a, Result: b})
	a.addMethod(&Method{Operator: ">=", Root: a, Operand: a, Result: b})
	a.addMethod(&Method{Operator: "==", Root: a, Operand: a, Result: b})

	// Bool logical methods
	b.addMethod(&Method{Operator: "==", Root: b, Operand: b, Result: b})
	b.addMethod(&Method{Operator: "&&", Root: b, Operand: b, Result: b})
	b.addMethod(&Method{Operator: "||", Root: b, Operand: b, Result: b})

	// Int arithmetic and comparison methods
	i.addMethod(&Method{Operator: "+", Root: i, Operand: i, Result: i})
	i.addMethod(&Method{Operator: "-", Root: i, Operand: i, Result: i})
	i.addMethod(&Method{Operator: "*", Root: i, Operand: i, Result: i})
	i.addMethod(&Method{Operator: "/", Root: i, Operand: i, Result: d})
	i.addMethod(&Method{Operator: "<", Root: i, Operand: i, Result: b})
	i.addMethod(&Method{Operator: "<=", Root: i, Operand: i, Result: b})
	i.addMethod(&Method{Operator: ">", Root: i, Operand: i, Result: b})
	i.addMethod(&Method{Operator: ">=", Root: i, Operand: i, Result: b})
	i.addMethod(&Method{Operator: "==", Root: i, Operand: i, Result: b})

	// Dec arithmetic and comparison methods
	d.addMethod(&Method{Operator: "+", Root: d, Operand: d, Result: d})
	d.addMethod(&Method{Operator: "-", Root: d, Operand: d, Result: d})
	d.addMethod(&Method{Operator: "*", Root: d, Operand: d, Result: d})
	d.addMethod(&Method{Operator: "/", Root: d, Operand: d, Result: d})
	d.addMethod(&Method{Operator: "<", Root: d, Operand: d, Result: b})
	d.addMethod(&Method{Operator: "<=", Root: d, Operand: d, Result: b})
	d.addMethod(&Method{Operator: ">", Root: d, Operand: d, Result: b})
	d.addMethod(&Method{Operator: ">=", Root: d, Operand: d, Result: b})
	d.addMethod(&Method{Operator: "==", Root: d, Operand: d, Result: b})

	// Str methods
	d.addMethod(&Method{Operator: "==", Root: s, Operand: s, Result: b})

	table.addType(a)
	table.addType(s)
	table.addType(d)
	table.addType(i)
	table.addType(b)

	return table
}

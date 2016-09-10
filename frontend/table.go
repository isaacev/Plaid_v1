package frontend

type typeTable struct {
	Table   map[string]typeTableEntry
	builtin builtinList
}

type typeTableEntry struct {
	Type Type
}

type builtinList struct {
	Any  *AnyType
	Func *TypeOperator
	Str  *TypeOperator
	Dec  *TypeOperator
	Int  *TypeOperator
	Bool *TypeOperator
}

func (table *typeTable) getNamedType(name string) (exists bool, typeRef Type) {
	if entry, ok := table.Table[name]; ok {
		return true, entry.Type
	}

	return false, nil
}

func (table *typeTable) addNamedType(name string, _type Type) {
	table.Table[name] = typeTableEntry{
		Type: _type,
	}
}

func newTypeTable() *typeTable {
	table := &typeTable{
		Table:   make(map[string]typeTableEntry),
		builtin: builtinList{},
	}

	a := &AnyType{}
	f := &TypeOperator{"Func", []Type{}, []*Method{}}
	s := &TypeOperator{"Str", []Type{}, []*Method{}}
	d := &TypeOperator{"Dec", []Type{}, []*Method{}}
	i := &TypeOperator{"Int", []Type{}, []*Method{}}
	b := &TypeOperator{"Bool", []Type{}, []*Method{}}

	// Any logical methods
	a.AddMethod(&Method{"<", a, a, b})
	a.AddMethod(&Method{"<=", a, a, b})
	a.AddMethod(&Method{">", a, a, b})
	a.AddMethod(&Method{">=", a, a, b})
	a.AddMethod(&Method{"==", a, a, b})

	// Str methods
	s.AddMethod(&Method{"==", s, s, b})
	s.AddMethod(&Method{"++", s, s, s})

	// Dec arithmetic and comparison methods
	d.AddMethod(&Method{"+", d, d, d})
	d.AddMethod(&Method{"-", d, d, d})   // binary addition
	d.AddMethod(&Method{"-", d, nil, d}) // unary negation
	d.AddMethod(&Method{"*", d, d, d})
	d.AddMethod(&Method{"/", d, d, d})
	d.AddMethod(&Method{"<", d, d, b})
	d.AddMethod(&Method{"<=", d, d, b})
	d.AddMethod(&Method{">", d, d, b})
	d.AddMethod(&Method{">=", d, d, b})
	d.AddMethod(&Method{"==", d, d, b})

	// Int arithmetic and comparison methods
	i.AddMethod(&Method{"+", i, i, i})
	i.AddMethod(&Method{"-", i, i, i})   // binary addition
	i.AddMethod(&Method{"-", i, nil, i}) // unary negation
	i.AddMethod(&Method{"*", i, i, i})
	i.AddMethod(&Method{"/", i, i, d})
	i.AddMethod(&Method{"<", i, i, b})
	i.AddMethod(&Method{"<=", i, i, b})
	i.AddMethod(&Method{">", i, i, b})
	i.AddMethod(&Method{">=", i, i, b})
	i.AddMethod(&Method{"==", i, i, b})

	// Bool logical methods
	b.AddMethod(&Method{"==", b, b, b})
	b.AddMethod(&Method{"&&", b, b, b})
	b.AddMethod(&Method{"||", b, b, b})

	table.builtin.Any = a
	table.addNamedType("Any", a)

	table.builtin.Bool = b
	table.addNamedType("Bool", b)

	table.builtin.Int = i
	table.addNamedType("Int", i)

	table.builtin.Dec = d
	table.addNamedType("Dec", d)

	table.builtin.Str = s
	table.addNamedType("Str", s)

	table.builtin.Func = f
	table.addNamedType("Func", f)

	return table
}

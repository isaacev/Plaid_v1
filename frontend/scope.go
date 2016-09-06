package frontend

import (
	"github.com/isaacev/Plaid/source"
)

// Scope represents the variable and type environment available at a point in
// a program's AST. This includes type data stored in the type table and
// variable names/type-signatures stored in the variables table. All scopes
// (except the global scope) have a parent scope for non-local symbol lookup
type Scope struct {
	File                *source.File
	Parent              *Scope
	types               *typeTable
	variables           map[string]definitionRecord
	upvalues            map[string]*UpvalueRecord
	registeredVariables []string
	registeredUpvalues  []string
	returns             []ReturnRecord
}

type definitionRecord struct {
	_type Type
	where definition
}

type definition struct {
	wholeDef  source.Span
	paramDefs []source.Span
	returnDef source.Span
}

type UpvalueRecord struct {
	Name          string
	LocalToParent bool
	LookupIndex   int
}

type LocalRecord struct {
	Name        string
	IsParameter bool
	LookupIndex int
}

type ReturnRecord struct {
	Type Type
	Span source.Span
}

func (s *Scope) registerLocalVariable(name string, _type Type, def definition) {
	s.registeredVariables = append(s.registeredVariables, name)
	s.variables[name] = definitionRecord{
		_type: _type,
		where: def,
	}
}

func (s *Scope) registerUpvalue(name string) (upvalueOffset int) {
	// Check if the upvalue has already been registered. If it has, return the
	// lookup index of that registration
	for i, upvalueName := range s.registeredUpvalues {
		if upvalueName == name {
			return i
		}
	}

	s.upvalues[name] = &UpvalueRecord{Name: name}
	upvalueOffset = len(s.registeredUpvalues)
	s.registeredUpvalues = append(s.registeredUpvalues, name)

	if s.Parent == nil {
		panic("cannot use upvalue in global scope")
	}

	if s.Parent.lookupLocalVariable(name) != nil {
		s.upvalues[name].LocalToParent = true

		for i, varName := range s.Parent.registeredVariables {
			if varName == name {
				s.upvalues[name].LookupIndex = i
				break
			}
		}
	} else {
		s.upvalues[name].LocalToParent = false
		s.upvalues[name].LookupIndex = s.Parent.registerUpvalue(name)
	}

	return upvalueOffset
}

func (s *Scope) lookupLocalVariable(name string) (_type Type) {
	if local, ok := s.variables[name]; ok {
		return local._type
	}

	return nil
}

func (s *Scope) lookupLocalVariableDefinition(name string) (exists bool, def definition) {
	if local, ok := s.variables[name]; ok {
		return true, local.where
	}

	return false, definition{}
}

func (s *Scope) lookupVariable(name string) (_type Type, isLocal bool) {
	if local, ok := s.variables[name]; ok {
		return local._type, true
	}

	if s.Parent != nil {
		_type, _ := s.Parent.lookupVariable(name)
		return _type, false
	}

	return nil, false
}

func newGlobalScope(file *source.File) *Scope {
	return &Scope{
		File:      file,
		types:     newTypeTable(),
		variables: make(map[string]definitionRecord),
		upvalues:  make(map[string]*UpvalueRecord),
	}
}

func (s *Scope) subScope() *Scope {
	return &Scope{
		Parent:    s,
		File:      s.File,
		types:     s.types,
		variables: make(map[string]definitionRecord),
		upvalues:  make(map[string]*UpvalueRecord),
	}
}

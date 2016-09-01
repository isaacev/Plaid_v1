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
	typeTable           *typeTable
	variables           map[string]*Signature
	upvalues            map[string]*UpvalueRecord
	registeredVariables []string
	registeredUpvalues  []string
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

func (s *Scope) registerLocalVariable(name string, sig *Signature) {
	s.registeredVariables = append(s.registeredVariables, name)
	s.variables[name] = sig
}

func (s *Scope) registerUpvalue(name string) (upvalueOffset int) {
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

func (s *Scope) lookupLocalVariable(name string) (sig *Signature) {
	if sig, ok := s.variables[name]; ok {
		return sig
	}

	return nil
}

func (s *Scope) lookupVariable(name string) (sig *Signature, isLocal bool) {
	if sig, ok := s.variables[name]; ok {
		return sig, true
	}

	if s.Parent != nil {
		sig, _ = s.Parent.lookupVariable(name)
		return sig, false
	}

	return nil, false
}

func newGlobalScope(file *source.File) *Scope {
	return &Scope{
		File:      file,
		typeTable: newTypeTable(),
		variables: make(map[string]*Signature),
		upvalues:  make(map[string]*UpvalueRecord),
	}
}

func (s *Scope) subScope() *Scope {
	return &Scope{
		Parent:    s,
		File:      s.File,
		typeTable: s.typeTable,
		variables: make(map[string]*Signature),
		upvalues:  make(map[string]*UpvalueRecord),
	}
}

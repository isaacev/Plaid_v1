package frontend

import (
	"math"

	"github.com/isaacev/Plaid/source"
)

// Type represents a type defined in a program. All types (except the Root type)
// have a parent for the purpose of type coersion. Types can be tagged with the
// line/column location of its definition
type Type struct {
	Name       string
	Parent     *Type
	Definition source.Span
}

// isDescendantOf returns true if "t2" is an ancestor on the inheritance tree
// than "t"
func (t *Type) isDescendantOf(t2 *Type) bool {
	if t2 == t {
		return true
	} else if t.Parent == nil {
		return false
	}

	return t.Parent.isDescendantOf(t2)
}

// getAncestry returns a slice of Types representing all the ancestor Types,
// with the last type being the Root Type
func (t *Type) getAncestry() (ancestry []*Type) {
	if t.Parent != nil {
		ancestry = t.Parent.getAncestry()
	}

	return append(ancestry, t)
}

// Signature includes fields for an expression's input types and a single output
// type. Signatures can also be tagged with the line/column information for
// where it was defined
type Signature struct {
	Output     *Type
	Inputs     []*Type
	Definition source.Span
}

// inheritanceTree represents the relationships between Types. The tree
// structure is used to determine how disparate types can be coerced into a
// single shared ancestor. All types can be coerced to the Root Type
type inheritanceTree struct {
	RootType *Type
	Table    map[string]*Type
}

// addType adds a new Type to the given inheritance tree. If the type is the
// first to be added to the tree, it becomes the Root Type. All types are stored
// in both a singly-linked child->parent tree structure and a map of
// type names-> type structs. The map provides a system for arbitrary type node
// lookup by type name
func (tree *inheritanceTree) addType(t *Type) {
	if tree.RootType == nil {
		tree.RootType = t
	}

	tree.Table[t.Name] = t
}

// lowestCommonAncestor takes two nodes on an inheritance tree and returns the
// lowest node on the tree that shares both given nodes as offspring
func (tree *inheritanceTree) lowestCommonAncestor(t1 *Type, t2 *Type) (t *Type) {
	t1Ancestors := t1.getAncestry()
	t2Ancestors := t2.getAncestry()

	lenT1Ancestry := float64(len(t1Ancestors))
	lenT2Ancestry := float64(len(t2Ancestors))

	min := int(math.Min(lenT1Ancestry, lenT2Ancestry))
	var commonAncestor *Type

	for i := 0; i < min; i++ {
		if t1Ancestors[i] == t2Ancestors[i] {
			commonAncestor = t1Ancestors[i]
		} else {
			break
		}
	}

	return commonAncestor
}

// newInheritanceTree builds a new tree structure and populates it with built-in
// types including the relationships between those built-in types
func newInheritanceTree() *inheritanceTree {
	tree := &inheritanceTree{
		RootType: nil,
		Table:    make(map[string]*Type),
	}

	o := Type{Name: "Object", Parent: nil}
	s := Type{Name: "String", Parent: &o}
	d := Type{Name: "Decimal", Parent: &o}
	i := Type{Name: "Integer", Parent: &d}
	b := Type{Name: "Boolean", Parent: &o}

	tree.addType(&o)
	tree.addType(&s)
	tree.addType(&d)
	tree.addType(&i)
	tree.addType(&b)

	return tree
}

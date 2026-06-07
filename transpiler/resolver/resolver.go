package resolver

import (
	"fmt"

	"github.com/nahmanmate/goanna/transpiler/ast"
)

type Variant struct {
	Name        string
	PayloadType string
	IsAtom      bool
}

type SymbolTable struct {
	Unions         map[string][]Variant // union name → ordered variants
	VariantToUnion map[string]string    // variant name → union name
	UsesAtom       bool                 // true if any union has atom variants
}

func Build(file *ast.File) (*SymbolTable, error) {
	tbl := &SymbolTable{
		Unions:         make(map[string][]Variant),
		VariantToUnion: make(map[string]string),
	}
	for _, item := range file.Items {
		decl, ok := item.(ast.UnionDecl)
		if !ok {
			continue
		}
		var variants []Variant
		for _, vg := range decl.Variants {
			for _, name := range vg.Names {
				if existing, clash := tbl.VariantToUnion[name]; clash {
					return nil, fmt.Errorf("variant %q already declared in union %q", name, existing)
				}
				isAtom := vg.Type == "atom"
				v := Variant{
					Name:        name,
					PayloadType: vg.Type,
					IsAtom:      isAtom,
				}
				if isAtom {
					tbl.UsesAtom = true
				}
				variants = append(variants, v)
				tbl.VariantToUnion[name] = decl.Name
			}
		}
		tbl.Unions[decl.Name] = variants
	}
	return tbl, nil
}

// LookupVariant finds a variant by name across all unions.
func (tbl *SymbolTable) LookupVariant(name string) (Variant, string, bool) {
	unionName, ok := tbl.VariantToUnion[name]
	if !ok {
		return Variant{}, "", false
	}
	for _, v := range tbl.Unions[unionName] {
		if v.Name == name {
			return v, unionName, true
		}
	}
	return Variant{}, "", false
}

// TailIdent returns the last dot-separated segment of an expression.
// "greg.gender" → "gender", "x" → "x"
func TailIdent(expr string) string {
	for i := len(expr) - 1; i >= 0; i-- {
		if expr[i] == '.' {
			return expr[i+1:]
		}
	}
	return expr
}

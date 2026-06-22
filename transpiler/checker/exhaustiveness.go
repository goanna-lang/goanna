package checker

import (
	"fmt"

	"github.com/goanna-lang/goanna/transpiler/ast"
	"github.com/goanna-lang/goanna/transpiler/resolver"
)

// CheckError is a structured diagnostic from the union checker.
type CheckError struct {
	ByteOffset int    // byte offset of the 'switch' keyword (sw.Line)
	Kind       string // "non_exhaustive" | "unknown_variant"
	Message    string
}

func (e *CheckError) Error() string { return e.Message }

// Check validates all union switches in the file.
// Returns structured errors (non-exhaustive switches, unknown variants).
func Check(file *ast.File, tbl *resolver.SymbolTable) []*CheckError {
	var errs []*CheckError
	for _, item := range file.Items {
		sw, ok := item.(ast.UnionSwitch)
		if !ok {
			continue
		}
		errs = append(errs, checkSwitch(sw, tbl)...)
	}
	return errs
}

// InferUnionName resolves the union type for a switch.
// First tries the tail identifier of the subject expression (works when field
// name matches type name, e.g. "greg.gender" → "gender").
// Falls back to looking up the first case label in the symbol table.
func InferUnionName(sw ast.UnionSwitch, tbl *resolver.SymbolTable) string {
	tail := resolver.TailIdent(sw.Subject)
	if _, ok := tbl.Unions[tail]; ok {
		return tail
	}
	// Infer from first case label variant.
	for _, uc := range sw.Cases {
		for _, name := range uc.VariantNames {
			if u, ok := tbl.VariantToUnion[name]; ok {
				return u
			}
		}
	}
	return tail
}

func checkSwitch(sw ast.UnionSwitch, tbl *resolver.SymbolTable) []*CheckError {
	var errs []*CheckError

	unionName := InferUnionName(sw, tbl)
	variants, known := tbl.Unions[unionName]
	if !known {
		// Could be a non-union type — skip (the compiler will catch real errors).
		return nil
	}

	covered := make(map[string]bool)
	for _, uc := range sw.Cases {
		for _, name := range uc.VariantNames {
			if _, _, exists := tbl.LookupVariant(name); !exists {
				errs = append(errs, &CheckError{
					ByteOffset: sw.Line,
					Kind:       "unknown_variant",
					Message:    fmt.Sprintf("line %d: case %q is not a variant of %s", sw.Line, name, unionName),
				})
				continue
			}
			covered[name] = true
		}
	}

	// default = opt out of exhaustiveness
	if sw.HasDefault {
		return errs
	}

	var missing []string
	for _, v := range variants {
		if !covered[v.Name] {
			missing = append(missing, v.Name)
		}
	}
	if len(missing) > 0 {
		errs = append(errs, &CheckError{
			ByteOffset: sw.Line,
			Kind:       "non_exhaustive",
			Message:    fmt.Sprintf("line %d: switch on %s is non-exhaustive: missing cases %v", sw.Line, unionName, missing),
		})
	}

	return errs
}

package ast

// File is the top-level result of parsing a .union.go file.
// Items are in source order: OpaqueChunks interleaved with UnionDecls and UnionSwitches.
type File struct {
	Items []Item
}

type Item interface{ item() }

// OpaqueChunk is verbatim source text the transpiler does not interpret.
type OpaqueChunk struct {
	Text string
}

func (OpaqueChunk) item() {}

// UnionDecl represents: type <Name> union { <variants> }
type UnionDecl struct {
	Line     int
	Name     string
	Variants []VariantGroup
}

func (UnionDecl) item() {}

// VariantGroup is one line in a union block.
// Names=["Male","Female"], Type="atom"  or  Names=["config1"], Type="normalConfig"
type VariantGroup struct {
	Names []string
	Type  string
}

// UnionSwitch represents a switch statement using .(union).
type UnionSwitch struct {
	Line       int
	BindVar    string      // empty if no binding variable
	Subject    string      // expression before .(union), e.g. "greg.gender"
	Cases      []UnionCase
	HasDefault bool
	DefaultBody string
}

func (UnionSwitch) item() {}

// UnionCase is one arm of a union switch.
type UnionCase struct {
	VariantNames []string
	Body         string // verbatim source for the case body
}

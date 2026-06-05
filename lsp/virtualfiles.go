package lsp

import (
	"net/url"
	"strings"
	"sync"

	"github.com/nahmanmate/gounion/ast"
	"github.com/nahmanmate/gounion/checker"
	"github.com/nahmanmate/gounion/resolver"
)

// VirtualFile holds the transpilation state for one open .union.go file.
type VirtualFile struct {
	SourceURI   string
	GenURI      string
	SourceBytes []byte
	GenBytes    []byte                // nil if parse/resolve error
	SourceMap   *SourceMap            // nil if transpile failed
	ASTFile     *ast.File
	SymTable    *resolver.SymbolTable
	CheckErrors []*checker.CheckError
	ParseError  error // non-nil = entire transpile failed
}

// Store manages open VirtualFile instances.
type Store struct {
	mu         sync.RWMutex
	bySource   map[string]*VirtualFile
	byGen      map[string]*VirtualFile
	goplsDiags map[string][]Diagnostic // sourceURI → latest translated gopls diagnostics
}

func NewStore() *Store {
	return &Store{
		bySource:   make(map[string]*VirtualFile),
		byGen:      make(map[string]*VirtualFile),
		goplsDiags: make(map[string][]Diagnostic),
	}
}

// IsUnionFile reports whether uri points to a .union.go file.
func (s *Store) IsUnionFile(uri string) bool {
	u, err := url.Parse(uri)
	if err != nil {
		return strings.HasSuffix(uri, ".union.go")
	}
	return strings.HasSuffix(u.Path, ".union.go")
}

func (s *Store) Get(sourceURI string) (*VirtualFile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.bySource[sourceURI]
	return v, ok
}

func (s *Store) GetByGenURI(genURI string) (*VirtualFile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.byGen[genURI]
	return v, ok
}

func (s *Store) Set(vf *VirtualFile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bySource[vf.SourceURI] = vf
	if vf.GenURI != "" {
		s.byGen[vf.GenURI] = vf
	}
}

func (s *Store) Delete(sourceURI string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if vf, ok := s.bySource[sourceURI]; ok {
		delete(s.byGen, vf.GenURI)
		delete(s.bySource, sourceURI)
	}
	delete(s.goplsDiags, sourceURI)
}

// StoreGoplsDiags atomically stores translated gopls diagnostics for sourceURI.
func (s *Store) StoreGoplsDiags(sourceURI string, diags []Diagnostic) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.goplsDiags[sourceURI] = diags
}

// GetGoplsDiags returns the latest gopls diagnostics for sourceURI (nil if none).
func (s *Store) GetGoplsDiags(sourceURI string) []Diagnostic {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.goplsDiags[sourceURI]
}

// sourceToGenURI converts a .union.go URI to the corresponding virtual .go URI.
func sourceToGenURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return strings.TrimSuffix(uri, ".union.go") + ".go"
	}
	u.Path = strings.TrimSuffix(u.Path, ".union.go") + ".go"
	return u.String()
}

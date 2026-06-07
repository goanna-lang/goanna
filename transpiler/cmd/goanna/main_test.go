package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProcessFileEmitsGoannTypes verifies that processFile writes goanna_types.go
// when the source contains atom union variants.
func TestProcessFileEmitsGoannTypes(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "example.goa")
	if err := os.WriteFile(src, []byte("package mypkg\n\ntype color union { Red, Blue atom }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cli.Output = ""
	cli.Check = false

	goannaDirs := make(map[string]bool)
	if err := processFile(src, goannaDirs); err != nil {
		t.Fatalf("processFile: %v", err)
	}

	goPath := filepath.Join(dir, "example.go")
	goContent, err := os.ReadFile(goPath)
	if err != nil {
		t.Fatalf("read example.go: %v", err)
	}
	if strings.Contains(string(goContent), "type atom struct{}") {
		t.Error("example.go should not contain 'type atom struct{}'")
	}

	typesPath := filepath.Join(dir, "goanna_types.go")
	typesContent, err := os.ReadFile(typesPath)
	if err != nil {
		t.Fatalf("goanna_types.go not created: %v", err)
	}
	got := string(typesContent)
	if !strings.Contains(got, "package mypkg") {
		t.Errorf("goanna_types.go missing package declaration: %s", got)
	}
	if !strings.Contains(got, "type atom struct{}") {
		t.Errorf("goanna_types.go missing atom type: %s", got)
	}
}

// TestProcessFileNoAtomNoGoannTypes verifies that goanna_types.go is NOT created
// when the source has no atom variants.
func TestProcessFileNoAtomNoGoannTypes(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "payload.goa")
	content := "package mypkg\n\ntype redConfig struct{ r int }\ntype color union { red redConfig }\n"
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cli.Output = ""
	cli.Check = false

	goannaDirs := make(map[string]bool)
	if err := processFile(src, goannaDirs); err != nil {
		t.Fatalf("processFile: %v", err)
	}

	typesPath := filepath.Join(dir, "goanna_types.go")
	if _, err := os.Stat(typesPath); err == nil {
		t.Error("goanna_types.go should not be created for payload-only unions")
	}
}

// TestProcessFileGoannTypesDeduplicated verifies that goanna_types.go is written only once
// when multiple .goa files in the same directory both use atom variants.
func TestProcessFileGoannTypesDeduplicated(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.goa", "b.goa"} {
		src := filepath.Join(dir, name)
		if err := os.WriteFile(src, []byte("package mypkg\n\ntype status union { Active, Inactive atom }\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cli.Output = ""
	cli.Check = false

	goannaDirs := make(map[string]bool)
	for _, name := range []string{"a.goa", "b.goa"} {
		if err := processFile(filepath.Join(dir, name), goannaDirs); err != nil {
			t.Fatalf("processFile(%s): %v", name, err)
		}
	}

	typesPath := filepath.Join(dir, "goanna_types.go")
	if _, err := os.Stat(typesPath); err != nil {
		t.Fatalf("goanna_types.go should exist: %v", err)
	}
	// Written once: goannaDirs should have exactly one entry.
	if len(goannaDirs) != 1 {
		t.Errorf("expected 1 entry in goannaDirs, got %d", len(goannaDirs))
	}
}

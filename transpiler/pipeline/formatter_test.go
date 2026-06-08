package pipeline

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// resetFormatter saves and restores formatter state around a test.
func resetFormatter(t *testing.T) {
	t.Helper()
	was := struct {
		enabled    bool
		useGofumpt bool
		exe        string
	}{formatterEnabled, useGofumpt, gofumptExe}
	t.Cleanup(func() {
		formatterEnabled = was.enabled
		useGofumpt = was.useGofumpt
		gofumptExe = was.exe
	})
}

// mockStdin replaces os.Stdin with a pipe pre-loaded with data, restoring on cleanup.
func mockStdin(t *testing.T, data string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig; _ = r.Close() })
}

func TestEnableGofmt(t *testing.T) {
	resetFormatter(t)
	formatterEnabled = false
	useGofumpt = true
	EnableGofmt()
	if !formatterEnabled {
		t.Error("formatterEnabled not set")
	}
	if useGofumpt {
		t.Error("useGofumpt should be cleared by EnableGofmt")
	}
}

func TestEnableGofumpt_available(t *testing.T) {
	resetFormatter(t)
	path, err := exec.LookPath("gofumpt")
	if err != nil {
		t.Skip("gofumpt not in PATH")
	}
	EnableGofumpt()
	if !formatterEnabled {
		t.Error("formatterEnabled not set")
	}
	if !useGofumpt {
		t.Error("useGofumpt not set")
	}
	if gofumptExe != path {
		t.Errorf("gofumptExe = %q, want %q", gofumptExe, path)
	}
}

func TestEnableGofumpt_decline(t *testing.T) {
	resetFormatter(t)
	// Hide gofumpt from PATH and GOPATH/bin so the prompt fires.
	t.Setenv("PATH", "/nonexistent")
	t.Setenv("GOPATH", t.TempDir())
	mockStdin(t, "n\n")
	// Swallow the prompt written to stderr.
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	origStderr := os.Stderr
	os.Stderr = devNull
	t.Cleanup(func() { os.Stderr = origStderr; _ = devNull.Close() })

	EnableGofumpt()

	if !formatterEnabled {
		t.Error("formatterEnabled should be set even on decline (fallback to gofmt)")
	}
	if useGofumpt {
		t.Error("useGofumpt should not be set when user declines install")
	}
}

func TestEnableGofumpt_pipedStdin(t *testing.T) {
	resetFormatter(t)
	// Hide gofumpt so the install-prompt path would normally fire.
	t.Setenv("PATH", "/nonexistent")
	t.Setenv("GOPATH", t.TempDir())
	// Simulate piped source: `cat foo.goa | goanna --gofumpt`
	mockStdin(t, "package main\nfunc main() {}\n")

	EnableGofumpt()

	// Should fall back to gofmt without touching stdin.
	if !formatterEnabled {
		t.Error("formatterEnabled should be set (fallback to gofmt)")
	}
	if useGofumpt {
		t.Error("useGofumpt should not be set when stdin is a pipe")
	}
	// Piped source must still be readable — not consumed by the prompt.
	remaining, err := io.ReadAll(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) == 0 {
		t.Error("stdin was consumed by EnableGofumpt; piped source would be lost")
	}
}

func TestFormatEmitted_disabled(t *testing.T) {
	resetFormatter(t)
	formatterEnabled = false
	src := []byte("package main\nfunc   main()  { }")
	got, err := formatEmitted(src, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, src) {
		t.Error("expected src returned unchanged when formatter disabled")
	}
}

func TestFormatEmitted_gofmt(t *testing.T) {
	resetFormatter(t)
	formatterEnabled = true
	useGofumpt = false
	src := []byte("package main\nfunc main(){\n}")
	got, err := formatEmitted(src, "test")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(got, src) {
		t.Error("expected gofmt to normalise formatting")
	}
	if !strings.Contains(string(got), "func main()") {
		t.Errorf("formatted output missing func main(): %s", got)
	}
}

func TestFormatEmitted_gofmt_invalidGo(t *testing.T) {
	resetFormatter(t)
	formatterEnabled = true
	useGofumpt = false
	_, err := formatEmitted([]byte("this is not go"), "myfile")
	if err == nil {
		t.Fatal("expected error for invalid Go, got nil")
	}
	if !strings.Contains(err.Error(), "myfile:") {
		t.Errorf("error missing name prefix: %v", err)
	}
	if !strings.Contains(err.Error(), "--- unformatted ---") {
		t.Errorf("error missing unformatted source dump: %v", err)
	}
}

func TestFormatEmitted_gofumpt(t *testing.T) {
	resetFormatter(t)
	path, err := exec.LookPath("gofumpt")
	if err != nil {
		t.Skip("gofumpt not in PATH")
	}
	formatterEnabled = true
	useGofumpt = true
	gofumptExe = path
	src := []byte("package main\nfunc main(){\n}")
	got, err := formatEmitted(src, "test")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(got, src) {
		t.Error("expected gofumpt to normalise formatting")
	}
	if !strings.Contains(string(got), "func main()") {
		t.Errorf("gofumpt output missing func main(): %s", got)
	}
}

func TestFormatEmitted_gofumpt_invalidGo(t *testing.T) {
	resetFormatter(t)
	path, err := exec.LookPath("gofumpt")
	if err != nil {
		t.Skip("gofumpt not in PATH")
	}
	formatterEnabled = true
	useGofumpt = true
	gofumptExe = path
	_, err = formatEmitted([]byte("not go code"), "myfile")
	if err == nil {
		t.Fatal("expected error for invalid Go input")
	}
	if !strings.Contains(err.Error(), "--- unformatted ---") {
		t.Errorf("error missing unformatted source dump: %v", err)
	}
}

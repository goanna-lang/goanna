package pipeline

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"strings"
)

const (
	gofumptModule  = "mvdan.cc/gofumpt"
	gofumptVersion = "v0.7.0"
)

var (
	formatterEnabled bool
	useGofumpt       bool
	gofumptExe       string
)

// EnableGofmt configures emitted files to be formatted with go/format.
func EnableGofmt() {
	formatterEnabled = true
}

// EnableGofumpt configures emitted files to be formatted with gofumpt.
// If gofumpt is not found, the user is prompted to install it; on failure or
// refusal, falls back to go/format.
func EnableGofumpt() {
	formatterEnabled = true
	if path, err := exec.LookPath("gofumpt"); err == nil {
		useGofumpt, gofumptExe = true, path
		return
	}
	if gopath, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		candidate := strings.TrimSpace(string(gopath)) + "/bin/gofumpt"
		if _, err := os.Stat(candidate); err == nil {
			useGofumpt, gofumptExe = true, candidate
			return
		}
	}

	fmt.Fprint(os.Stderr, "gofumpt not found. Install it? [y/N] ")
	var resp string
	fmt.Fscan(os.Stdin, &resp)
	if strings.ToLower(strings.TrimSpace(resp)) != "y" {
		return
	}

	cmd := exec.Command("go", "install", gofumptModule+"@"+gofumptVersion)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gofumpt install failed: %v\n", err)
		return
	}

	if path, err := exec.LookPath("gofumpt"); err == nil {
		useGofumpt, gofumptExe = true, path
		return
	}
	if gopath, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		candidate := strings.TrimSpace(string(gopath)) + "/bin/gofumpt"
		if _, err := os.Stat(candidate); err == nil {
			useGofumpt, gofumptExe = true, candidate
		}
	}
}

func formatEmitted(src []byte, name string) ([]byte, error) {
	if !formatterEnabled {
		return src, nil
	}
	if useGofumpt {
		cmd := exec.Command(gofumptExe)
		cmd.Stdin = bytes.NewReader(src)
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("%s: gofumpt: %w\n--- unformatted ---\n%s", name, err, src)
		}
		return out, nil
	}
	out, err := format.Source(src)
	if err != nil {
		return nil, fmt.Errorf("%s: format: %w\n--- unformatted ---\n%s", name, err, src)
	}
	return out, nil
}

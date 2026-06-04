package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

// GoplsProcess manages a running gopls subprocess.
type GoplsProcess struct {
	cmd    *exec.Cmd
	In     *Writer
	stdout *bufio.Reader
}

// SpawnGopls starts `gopls serve` and returns a handle to it.
// goplsPath may be empty, in which case FindGopls is used.
func SpawnGopls(ctx context.Context, goplsPath string) (*GoplsProcess, error) {
	if goplsPath == "" {
		goplsPath = FindGopls()
	}

	cmd := exec.CommandContext(ctx, goplsPath, "serve")

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("gopls stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("gopls stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("gopls stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start gopls: %w", err)
	}

	go func() {
		io.Copy(log.Writer(), stderrPipe) //nolint:errcheck
	}()

	return &GoplsProcess{
		cmd:    cmd,
		In:     NewWriter(stdinPipe),
		stdout: bufio.NewReader(stdoutPipe),
	}, nil
}

func (g *GoplsProcess) ReadMessage() (*Message, error) {
	return ReadMessage(g.stdout)
}

func (g *GoplsProcess) Wait() error {
	return g.cmd.Wait()
}

func (g *GoplsProcess) Kill() {
	if g.cmd.Process != nil {
		_ = g.cmd.Process.Kill()
	}
}

// Shutdown sends shutdown + exit to gopls so it can clean up.
func (g *GoplsProcess) Shutdown() {
	id, _ := json.Marshal(9999999)
	_ = g.In.Write(&Message{Method: "shutdown", ID: json.RawMessage(id)})
	_ = g.In.Write(&Message{Method: "exit"})
}

// FindGopls returns a path to the gopls binary.
func FindGopls() string {
	if p, err := exec.LookPath("gopls"); err == nil {
		return p
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return gopath + "/bin/gopls"
	}
	home, _ := os.UserHomeDir()
	return home + "/go/bin/gopls"
}

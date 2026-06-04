package lsp

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
)

// Config holds options for the LSP proxy server.
type Config struct {
	GoplsPath string // path to gopls binary; defaults to PATH lookup
}

// Run starts the LSP proxy, reading from stdin and writing to stdout.
// It blocks until stdin is closed (editor disconnected) or ctx is cancelled.
func Run(ctx context.Context, cfg Config) error {
	logger := log.New(os.Stderr, "[gounion-lsp] ", log.LstdFlags)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	goplsProc, err := SpawnGopls(ctx, cfg.GoplsPath)
	if err != nil {
		return fmt.Errorf("spawn gopls: %w", err)
	}
	defer goplsProc.Kill()

	editorOut := NewWriter(os.Stdout)
	store := NewStore()
	proxy := NewProxy(store, editorOut, goplsProc, logger)

	// Pump gopls → proxy in background.
	goplsExitCh := make(chan error, 1)
	go func() {
		goplsExitCh <- goplsProc.Wait()
	}()

	goplsDone := make(chan struct{})
	go func() {
		defer close(goplsDone)
		for {
			msg, err := goplsProc.ReadMessage()
			if err != nil {
				select {
				case <-ctx.Done():
				default:
					logger.Printf("read from gopls: %v", err)
				}
				return
			}
			proxy.HandleGoplsMessage(msg)
		}
	}()

	// Pump editor → proxy (blocking, on stdin).
	editorIn := bufio.NewReader(os.Stdin)
	for {
		msg, err := ReadMessage(editorIn)
		if err != nil {
			// EOF = editor closed connection.
			break
		}
		proxy.HandleEditorMessage(msg)
	}

	// Editor closed; send graceful shutdown to gopls.
	goplsProc.Shutdown()
	cancel()
	<-goplsDone
	return nil
}

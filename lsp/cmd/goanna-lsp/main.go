package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nahmanmate/goanna/lsp"
)

func main() {
	cfg := lsp.Config{}
	for _, a := range os.Args[1:] {
		if p, ok := strings.CutPrefix(a, "--gopls="); ok {
			cfg.GoplsPath = p
		}
	}
	if err := lsp.Run(context.Background(), cfg); err != nil {
		fmt.Fprintf(os.Stderr, "goanna-lsp: %v\n", err)
		os.Exit(1)
	}
}

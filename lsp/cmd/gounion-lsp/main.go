package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nahmanmate/gounion/lsp"
)

func main() {
	cfg := lsp.Config{}
	for _, a := range os.Args[1:] {
		if p, ok := strings.CutPrefix(a, "--gopls="); ok {
			cfg.GoplsPath = p
		}
	}
	if err := lsp.Run(context.Background(), cfg); err != nil {
		fmt.Fprintf(os.Stderr, "gounion-lsp: %v\n", err)
		os.Exit(1)
	}
}

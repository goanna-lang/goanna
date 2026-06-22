package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/goanna-lang/goanna/lsp"
)

var version = "dev"

func main() {
	goplsPath := flag.String("gopls", "", "path to gopls binary")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("goanna-lsp", version)
		return
	}

	cfg := lsp.Config{
		GoplsPath: *goplsPath,
	}
	if err := lsp.Run(context.Background(), cfg); err != nil {
		fmt.Fprintf(os.Stderr, "goanna-lsp: %v\n", err)
		os.Exit(1)
	}
}

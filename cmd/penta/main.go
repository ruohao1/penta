package main

import (
	"os"

	"github.com/Ruohao1/penta/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

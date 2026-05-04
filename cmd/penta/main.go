package main

import (
	"fmt"
	"os"

	"github.com/ruohao1/penta/internal/apperr"
	"github.com/ruohao1/penta/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		if !apperr.IsReported(err) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

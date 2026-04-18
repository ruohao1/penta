package main

import (
	"github.com/ruohao1/penta/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		panic(err)
	}
}

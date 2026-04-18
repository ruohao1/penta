package main

import (
	"fmt"
	"os"

	"github.com/ruohao1/penta/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

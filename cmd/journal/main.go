package main

import (
	"os"

	"github.com/data-castle/journal/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args))
}

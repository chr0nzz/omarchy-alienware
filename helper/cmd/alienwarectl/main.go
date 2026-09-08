package main

import (
	"os"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}

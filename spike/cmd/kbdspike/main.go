package main

import (
	"os"

	"github.com/chr0nzz/omarchy-alienware/alienfx-spike/internal/kbdcli"
)

func main() {
	os.Exit(kbdcli.Run(os.Args[1:]))
}

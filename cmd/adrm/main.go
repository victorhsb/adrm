package main

import (
	"os"

	"github.com/victorhsb/adrm/internal/adrm"
)

func main() {
	os.Exit(adrm.Run(os.Args[1:], os.Stdout, os.Stderr))
}

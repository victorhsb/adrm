package main

import (
	"os"

	"github.com/victorhsb/canon/internal/canon"
)

func main() {
	os.Exit(canon.Run(os.Args[1:], os.Stdout, os.Stderr))
}

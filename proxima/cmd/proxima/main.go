package main

import (
	"os"

	"proxima/proxima/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}

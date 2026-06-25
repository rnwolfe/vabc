// Command vabc is the agent-cli-factory Go reference implementation.
//
// It implements the agent-CLI contract (references/contract.md) end-to-end against a
// trivial local-file "item" store, so the contract surface is provably correct and
// testable. cli-scaffold copies this tree and substitutes the template tokens documented
// in references/templates/go/TEMPLATE.md.
package main

import (
	"os"

	"github.com/rnwolfe/vabc/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// Command rapibase is the Rapibase CLI: log in, link a folder to a project,
// write its .env, connect your coding agent and publish, from the terminal.
package main

import (
	"os"

	"github.com/rapibase/cli/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}

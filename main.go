// The two verbs this repository is driven by, and no script beside them. The
// workflow runs the same verb a contributor runs, so there is one procedure
// rather than two, and a leg added to the gate is added in one place and
// reaches both routes without either being edited.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/Flowfin/site/internal/gate"
	"github.com/Flowfin/site/internal/hygiene"
	"github.com/Flowfin/site/internal/invariant"
	"github.com/Flowfin/site/internal/reproduce"
	"github.com/Flowfin/site/internal/site"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// run is separate from main so that a test can drive a verb and read what it
// wrote without the process exiting underneath it.
func run(args []string, out, errOut io.Writer) error {
	if len(args) == 0 {
		usage(errOut)
		return errors.New("a verb is required")
	}

	switch args[0] {
	case "build":
		if len(args) != 1 {
			usage(errOut)
			return errors.New("build takes no argument")
		}
		_, err := site.Build(".", site.OutputDir, out)
		return err
	case "ci":
		if len(args) != 1 {
			usage(errOut)
			return errors.New("ci takes no argument")
		}
		return gate.Run(".", out)
	case "invariants":
		if len(args) != 1 {
			usage(errOut)
			return errors.New("invariants takes no argument")
		}
		return invariant.Run(".", out)
	case "reproduce":
		if len(args) != 1 {
			usage(errOut)
			return errors.New("reproduce takes no argument")
		}
		return reproduce.Run(".", out)
	case "hygiene":
		return hygiene.Run(args[1:], out)
	default:
		usage(errOut)
		return fmt.Errorf("unknown verb %q", args[0])
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `Usage:
  go run . build   render the site into `+site.OutputDir+`/ and print what was written
  go run . ci      run the gate: every leg in order, stopping at the first failure
  go run . invariants
                   decide the rules a machine can read off the tree and the
                   output a build produces
  go run . reproduce
                   build twice and compare what came out, byte for byte
  go run . hygiene [-origin=internal|external] <base> <head>
                   judge the commit messages in a range
`)
}

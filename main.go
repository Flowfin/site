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

	"github.com/Flowfin/site/internal/bom"
	"github.com/Flowfin/site/internal/changelog"
	"github.com/Flowfin/site/internal/gate"
	"github.com/Flowfin/site/internal/hygiene"
	"github.com/Flowfin/site/internal/invariant"
	"github.com/Flowfin/site/internal/pins"
	"github.com/Flowfin/site/internal/reproduce"
	"github.com/Flowfin/site/internal/site"
	"github.com/Flowfin/site/internal/tokens"
	"github.com/Flowfin/site/internal/version"
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
	case "sbom":
		if len(args) != 1 {
			usage(errOut)
			return errors.New("sbom takes no argument")
		}
		// The document goes to the output stream and the report of what it
		// covered goes beside it, so redirecting the first into a file
		// leaves a reader with the second rather than with silence.
		return bom.Write(".", out, errOut)
	case "pins":
		if len(args) != 1 {
			usage(errOut)
			return errors.New("pins takes no argument")
		}
		// Deliberately not a leg of the gate. What it reads is three
		// registries over the network, so its verdict moves when somebody
		// else publishes rather than when this tree changes, and the half
		// that needs no network is decided by the suite and by the
		// invariant gate on every run.
		return pins.Run(".", pins.Registries, out)
	case "tokens":
		if len(args) != 1 {
			usage(errOut)
			return errors.New("tokens takes no argument")
		}
		// Deliberately not a leg of the gate, for the reason the pins verb
		// above is not one: what it reads is a file somebody else
		// publishes, so its verdict moves when they change it rather than
		// when this tree does. What the build needs from the copy is
		// decided on every run, because the build reads it.
		return tokens.Run(".", tokens.Publisher, out)
	case "version":
		if len(args) != 1 {
			usage(errOut)
			return errors.New("version takes no argument")
		}
		// The release run reads the version here rather than out of a
		// file, so the value the tag is built from is the one the build
		// and the bill of materials already carry. It prints the version
		// alone and on one line, because what reads it is a shell step
		// assigning it to a variable.
		_, err := fmt.Fprintln(out, version.Number)
		return err
	case "changelog":
		if len(args) != 1 {
			usage(errOut)
			return errors.New("changelog takes no argument")
		}
		// Deliberately not a leg of the gate. The version in the tree is
		// the one the next release carries, and the section describing it
		// is written when somebody decides to release rather than when
		// they bump a constant, so a leg would refuse every ordinary
		// change for a section nobody owes yet. The release run is where
		// the description is owed, and it runs this before it creates
		// anything.
		return changelog.Run(".", out)
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
  go run . sbom    write the bill of materials for what produces the published
                   bytes, as CycloneDX on the output stream
  go run . pins    compare every version in `+pins.File+` against the registry
                   that publishes it, and report without writing anything
  go run . tokens  compare the pinned copy in `+tokens.File+` against the
                   published token file, naming every value that differs
  go run . version print the version this repository releases under, alone on
                   one line, which is what the release run builds its tag from
  go run . changelog
                   refuse a version that `+changelog.File+` does not describe,
                   which is what the release run asks before it creates a tag
  go run . hygiene [-origin=internal|external] <base> <head>
                   judge the commit messages in a range
`)
}

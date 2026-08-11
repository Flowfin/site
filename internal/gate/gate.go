// Package gate is the whole of `go run . ci`.
//
// The legs run in the order they are declared and the run stops at the first
// failure, because a leg that runs after a failure is judging a tree somebody
// is already fixing. What that costs is that a run can end having examined
// less than the whole set, so the run prints which legs ran and names the ones
// it never reached. A partial run that printed only its failure would be read
// as a run that found one thing wrong and nothing else.
package gate

import (
	"fmt"
	"go/format"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Flowfin/site/internal/invariant"
	"github.com/Flowfin/site/internal/link"
	"github.com/Flowfin/site/internal/site"
)

// A leg is one thing the gate decides. Its report is the sentence printed
// beside its name when it passes, which is where a leg says it examined
// nothing rather than passing silently.
type leg struct {
	name string
	run  func(root string) (report string, err error)
}

func legs() []leg {
	return []leg{
		{"format", formatLeg},
		{"vet", vetLeg},
		{"test", testLeg},
		{"build", buildLeg},
		{"links", linksLeg},
		{"invariants", invariantsLeg},
	}
}

// Run executes every leg against the tree at root, in order, and stops at the
// first failure.
func Run(root string, log io.Writer) error {
	all := legs()

	names := make([]string, len(all))
	for i, l := range all {
		names[i] = l.name
	}
	fmt.Fprintf(log, "gate: %d legs, in order: %s\n", len(all), strings.Join(names, ", "))

	for i, l := range all {
		report, err := l.run(root)
		if err != nil {
			fmt.Fprintf(log, "  %s: FAILED\n", l.name)
			for _, line := range strings.Split(strings.TrimRight(err.Error(), "\n"), "\n") {
				fmt.Fprintf(log, "    %s\n", line)
			}
			notReached := names[i+1:]
			if len(notReached) == 0 {
				fmt.Fprintf(log, "%d of %d legs ran. The failure is in the last one.\n", i+1, len(all))
			} else {
				fmt.Fprintf(log, "%d of %d legs ran. Not reached: %s.\n",
					i+1, len(all), strings.Join(notReached, ", "))
			}
			return fmt.Errorf("gate: the %s leg refused this tree", l.name)
		}
		fmt.Fprintf(log, "  %s: %s\n", l.name, report)
	}

	fmt.Fprintf(log, "%d of %d legs ran. None was skipped.\n", len(all), len(all))
	return nil
}

// formatLeg refuses a Go file whose bytes differ from what gofmt would write.
// It compares in process rather than shelling out, so the answer does not
// depend on a gofmt binary being on the path beside the toolchain.
func formatLeg(root string) (string, error) {
	files, err := goFiles(root)
	if err != nil {
		return "", err
	}
	var bad []string
	for _, f := range files {
		src, err := os.ReadFile(filepath.Clean(f))
		if err != nil {
			return "", err
		}
		want, err := format.Source(src)
		if err != nil {
			return "", fmt.Errorf("%s does not parse: %v", rel(root, f), err)
		}
		if string(want) != string(src) {
			bad = append(bad, rel(root, f))
		}
	}
	if len(bad) > 0 {
		return "", fmt.Errorf("not gofmt-clean:\n%s", indent(bad))
	}
	if len(files) == 0 {
		return "no Go file in the tree, so this leg examined nothing", nil
	}
	return fmt.Sprintf("ok, %d file(s)", len(files)), nil
}

// vetLeg runs the toolchain's own suspicious-construct analysis.
func vetLeg(root string) (string, error) {
	out, err := goCmd(root, "vet", "./...")
	if err != nil {
		return "", fmt.Errorf("go vet refused:\n%s", indent(strings.Split(strings.TrimRight(out, "\n"), "\n")))
	}
	return "ok", nil
}

// testLeg runs the suite. A tree with no test file passes and says so: a leg
// that reported ok having run nothing is the shape this repository refuses
// everywhere else, and the harness that fills it is its own issue.
func testLeg(root string) (string, error) {
	files, err := goFiles(root)
	if err != nil {
		return "", err
	}
	n := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			n++
		}
	}
	if n == 0 {
		return "no test file in the tree, so this leg examined nothing", nil
	}
	out, err := goCmd(root, "test", "./...")
	if err != nil {
		return "", fmt.Errorf("go test refused:\n%s", indent(strings.Split(strings.TrimRight(out, "\n"), "\n")))
	}
	return fmt.Sprintf("ok, %d test file(s)", n), nil
}

// buildLeg renders the site into a directory it then throws away, so the gate
// answers whether the build works without leaving output behind that a reader
// might mistake for the result of `go run . build`.
func buildLeg(root string) (string, error) {
	tmp, err := os.MkdirTemp("", "site-gate-build-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	written, err := site.Build(root, filepath.Join(tmp, site.OutputDir), io.Discard)
	if err != nil {
		return "", fmt.Errorf("the build refused:\n    %v", err)
	}
	if len(written) == 0 {
		return "", fmt.Errorf("the build wrote no file, and a site of nothing is not a site")
	}
	return fmt.Sprintf("ok, %d file(s)", len(written)), nil
}

// linksLeg walks what the build produced and refuses a reference that resolves
// to nothing. It sits after the build leg because it has nothing to walk until
// the build works, and before the invariants leg because a page that points at a
// file nobody wrote is a broken site whatever else is true of its markup.
func linksLeg(root string) (string, error) {
	var log strings.Builder
	if err := link.Run(root, &log); err != nil {
		lines := strings.Split(strings.TrimRight(log.String(), "\n"), "\n")
		return "", fmt.Errorf("%v:\n%s", err, indent(lines))
	}
	// The last line the walk wrote is the one that says what it covered, and
	// it is that line rather than a count assembled here, so a run that
	// examined nothing says so in the walk's own words.
	lines := strings.Split(strings.TrimRight(log.String(), "\n"), "\n")
	return strings.TrimSpace(lines[len(lines)-1]), nil
}

// invariantsLeg decides the rules that can be read off the tree and off the
// output a build produces. The rows live in their own package rather than here,
// so the same set is decided by this leg and by the workflow that reports it
// under the name a rule on the branch can require. What the leg adds is that a
// contributor meets the rows by running one verb rather than by remembering a
// second one.
func invariantsLeg(root string) (string, error) {
	var log strings.Builder
	if err := invariant.Run(root, &log); err != nil {
		lines := strings.Split(strings.TrimRight(log.String(), "\n"), "\n")
		return "", fmt.Errorf("%v:\n%s", err, indent(lines))
	}
	rules, owed := len(invariant.Rules()), len(invariant.Owing())
	return fmt.Sprintf("ok, %d rule(s) decided, %d owed and not decided", rules, owed), nil
}

// goFiles lists the Go source in the tree. The output directory is skipped
// because it is not source, and the git directory because nothing in it is.
func goFiles(root string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", site.OutputDir:
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".go") {
			found = append(found, p)
		}
		return nil
	})
	return found, err
}

func goCmd(root string, args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func rel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(r)
}

func indent(lines []string) string {
	var b strings.Builder
	for i, l := range lines {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("    " + l)
	}
	return b.String()
}

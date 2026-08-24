// SPDX-License-Identifier: AGPL-3.0-or-later

// Package reproduce builds the site twice and compares what came out.
//
// The claim this repository wants to be able to make is that the bytes published
// at a tag can be produced again from the source at that tag. A claim like that
// is worth nothing unless something re-derives it, so it is re-derived rather
// than asserted.
//
// It reports under a name of its own rather than as a leg of the gate verb. A
// reproducibility failure and a compile failure are different problems with
// different repairs, and a reader of a red pull request should not have to open
// a log to tell which one they have.
package reproduce

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Flowfin/site/internal/site"
)

// Run builds the tree at root twice, into two directories it throws away, and
// refuses if the two differ anywhere.
func Run(root string, log io.Writer) error {
	return run(root, log, build)
}

// builder renders one build and reads back what it wrote. Run has the real one;
// the suite supplies its own, because a comparison can only be proved to notice a
// difference by being given one, and a build that reads only its source will not
// produce one on request.
type builder func(root, which string) (map[string][]byte, error)

func run(root string, log io.Writer, render builder) error {
	fmt.Fprintf(log, "reproduce: two builds of %s, compared byte for byte\n", filepath.ToSlash(root))

	first, err := render(root, "first")
	if err != nil {
		return err
	}
	second, err := render(root, "second")
	if err != nil {
		return err
	}

	names := union(first, second)
	if len(names) == 0 {
		return fmt.Errorf("both builds wrote nothing, and two empty directories are not a match")
	}

	var differ []string
	for _, n := range names {
		a, inA := first[n]
		b, inB := second[n]
		switch {
		case !inA:
			differ = append(differ, fmt.Sprintf("%s: only the second build wrote it", n))
		case !inB:
			differ = append(differ, fmt.Sprintf("%s: only the first build wrote it", n))
		case !bytes.Equal(a, b):
			differ = append(differ, fmt.Sprintf("%s: %d bytes against %d, and they are not the same bytes%s",
				n, len(a), len(b), firstDifference(a, b)))
		}
	}

	if len(differ) > 0 {
		fmt.Fprintf(log, "  %d of %d file(s) differ between the two builds\n", len(differ), len(names))
		for _, d := range differ {
			fmt.Fprintf(log, "    %s\n", d)
		}
		fmt.Fprintf(log, "The usual causes are a time read from the clock, a map walked in whatever order it came out in, and an absolute path from the build machine.\n")
		return fmt.Errorf("reproduce: %d of %d produced file(s) differ between two builds of one source", len(differ), len(names))
	}

	fmt.Fprintf(log, "  %d file(s), identical in both builds\n", len(names))
	return nil
}

// build renders into a temporary directory and reads back every file it wrote,
// keyed by its path inside the output. The directory is removed before this
// returns, so nothing is left for a later run to read by accident.
func build(root, which string) (map[string][]byte, error) {
	tmp, err := os.MkdirTemp("", "site-reproduce-"+which+"-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	out := filepath.Join(tmp, site.OutputDir)
	written, err := site.Build(root, out, io.Discard)
	if err != nil {
		return nil, fmt.Errorf("the %s build refused: %w", which, err)
	}

	label := filepath.ToSlash(out) + "/"
	files := make(map[string][]byte, len(written))
	for _, w := range written {
		rel := strings.TrimPrefix(filepath.ToSlash(w), label)
		b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		files[path.Join(site.OutputDir, rel)] = b
	}
	return files, nil
}

func union(a, b map[string][]byte) []string {
	seen := make(map[string]bool, len(a)+len(b))
	for n := range a {
		seen[n] = true
	}
	for n := range b {
		seen[n] = true
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// firstDifference points at the byte where the two copies part, so the repair
// starts at a line rather than at a whole file.
func firstDifference(a, b []byte) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return fmt.Sprintf(", first at line %d", bytes.Count(a[:i], []byte("\n"))+1)
		}
	}
	return ", one is a prefix of the other"
}

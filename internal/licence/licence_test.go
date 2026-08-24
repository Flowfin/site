// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the two readings this package makes: where a header may sit,
// and what counts as a source file at all.
//
// The fixtures are ordinary literals rather than base64. The convention in this
// tree spells a fixture in base64 where the exact bytes are the point and the
// tree's own rules would otherwise refuse the source carrying them; neither
// applies here, because what these fixtures hold is a header this repository
// wants on every file and a line the rule reads past.
package licence

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Where a header is read from, and where it is not. The last case is the one
// that decides the shape of the rule: a declaration below the code is a
// declaration nobody reading the first screen of a copied file meets.
func TestTheHeaderIsReadFromTheOpeningAndNowhereElse(t *testing.T) {
	for name, tc := range map[string]struct {
		body    string
		want    string
		carried bool
	}{
		"the first line": {
			body:    "// " + Header + "\n\npackage sample\n",
			want:    Identifier,
			carried: true,
		},
		"under an interpreter line": {
			body:    "#!/usr/bin/env bash\n# " + Header + "\n\nset -euo pipefail\n",
			want:    Identifier,
			carried: true,
		},
		"below a paragraph of comment": {
			body:    "// What this file is.\n//\n// " + Header + "\n\npackage sample\n",
			want:    Identifier,
			carried: true,
		},
		"the wrong identifier": {
			body:    "// " + Tag + " AGPL-3.0-only\n\npackage sample\n",
			want:    "AGPL-3.0-only",
			carried: true,
		},
		"no header at all": {
			body:    "// What this file is.\n\npackage sample\n",
			carried: false,
		},
		"below the code": {
			body:    "package sample\n\n// " + Header + "\n",
			carried: false,
		},
	} {
		got, ok := Declared([]byte(tc.body))
		if ok != tc.carried {
			t.Errorf("%s: Declared reported carried=%v, want %v", name, ok, tc.carried)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: Declared read %q, want %q", name, got, tc.want)
		}
	}
}

// What opens a source file and what does not. The page is the case that decides
// whether the rule using this can sit in a table beside rows that read produced
// markup: a body that is not source has to come back false, or a rule refusing
// a missing header would refuse every page the build wrote.
func TestASourceFileIsToldApartFromEverythingElse(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		want bool
	}{
		"a go file":                     {body: "package sample\n", want: true},
		"a go file under its comment":   {body: "// What this is.\n\npackage sample\n", want: true},
		"a go file with nothing but it": {body: "package sample", want: true},
		"a shell script":                {body: "#!/usr/bin/env bash\nset -e\n", want: true},
		"a produced page":               {body: "<!DOCTYPE html>\n<html lang=\"en\"></html>\n", want: false},
		"a json copy":                   {body: "{\"a\":1}\n", want: false},
		"a document":                    {body: "# A heading\n\nA paragraph.\n", want: false},
		"nothing at all":                {body: "", want: false},
	} {
		if got := IsSource([]byte(tc.body)); got != tc.want {
			t.Errorf("%s: IsSource reported %v, want %v", name, got, tc.want)
		}
	}
}

// The floor under IsSource, taken against the tree rather than against a
// fixture. The population the rule reads is cut by extension and this reading
// is cut by shape, so a file the second one does not recognise is a file the
// rule silently passes. That is the one failure neither reading can report on
// its own, and it is what this case is for.
func TestEverySourceFileThisTreeTracksIsRecognised(t *testing.T) {
	root := repoRoot(t)
	for _, name := range tracked(t, root, "*.go", "*.sh") {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if !IsSource(b) {
			t.Errorf("%s is tracked as a source file and IsSource does not recognise it, so the rule that reads the header passes it silently", name)
		}
	}
}

func tracked(t *testing.T, root string, patterns ...string) []string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"ls-files", "-z", "--"}, patterns...)...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("listing the tracked sources: %v", err)
	}
	var names []string
	for _, n := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if n != "" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		t.Fatalf("this tree tracks no source file matching %v, and the case would pass having read nothing", patterns)
	}
	return names
}

func repoRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("finding the repository root: %v", err)
	}
	return strings.TrimSpace(string(out))
}

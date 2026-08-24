// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the pinned copy and the comparison that says when it has
// fallen behind.
//
// Every case that needs a published file supplies one from a fetcher of its
// own, so the four states the run has to tell apart are all reachable without a
// network and without waiting for somebody else's host to be having a bad day.
// The one case that reads the real tree reads the copy this repository actually
// carries, and it asks whether that copy parses rather than what is in it.
//
// No case opens a window, binds a socket, reaches the network or needs anything
// that is not in the toolchain, so `go test ./...` is the whole harness.
package tokens

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// answer is a fetcher that hands back what a case wrote for it.
func answer(body string) Fetcher {
	return func() ([]byte, error) { return []byte(body), nil }
}

func fails(err error) Fetcher {
	return func() ([]byte, error) { return nil, err }
}

// pin writes a pinned copy under a temporary root and returns the root.
func pin(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, filepath.Dir(filepath.FromSlash(File)))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("preparing %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(File)), []byte(body), 0o644); err != nil {
		t.Fatalf("writing the pinned copy: %v", err)
	}
	return root
}

const oneGroup = `{"surface":{"tokens":{"ground":{"css-var":"--ground","dark":{"srgb":"#121216","alpha":1}}}}}`

// A path is what a difference is named by, so the flattening has to produce the
// name a reader would go looking for. The interesting one is the number: taken
// through a float, a limit of 1200 comes back as 1.2e+03, and a reader then has
// to decode a difference that is not there.
func TestReadNamesEveryValueByItsPathAndKeepsTheDigitsTheFileWasWrittenWith(t *testing.T) {
	values, err := Read([]byte(`{"budget":{"numbers":{"first-picture":{"limit":1200,"comparison":"below"}}},
		"font":{"sans":{"stack":["ui-sans-serif","system-ui"]}},"shape":{"radius":{"value":12.5}}}`))
	if err != nil {
		t.Fatalf("reading a token file that breaks nothing: %v", err)
	}

	for name, want := range map[string]string{
		"budget.numbers.first-picture.limit":      "1200",
		"budget.numbers.first-picture.comparison": "below",
		"font.sans.stack[0]":                      "ui-sans-serif",
		"font.sans.stack[1]":                      "system-ui",
		"shape.radius.value":                      "12.5",
	} {
		if got := values[name]; got != want {
			t.Errorf("%s read as %q, want %q", name, got, want)
		}
	}
	if len(values) != 5 {
		t.Errorf("%d value(s) read, want 5: %v", len(values), values)
	}
}

// Each refusal is the shape of a file somebody could actually commit, and each
// one is checked on its own so a message that stopped naming what was wrong
// shows up here rather than in a bisect.
func TestReadRefusesEachThingItDoesNotUnderstand(t *testing.T) {
	for _, c := range []struct {
		name, body, says string
	}{
		{"not JSON at all", "{", "is not JSON"},
		{"an array rather than the object of groups", `[{"radius":12}]`, "object of groups"},
		{"a document carrying no value", `{}`, "carries no value"},
		{"groups that are all empty", `{"surface":{},"type":{"roles":{}}}`, "carries no value"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Read([]byte(c.body))
			if err == nil {
				t.Fatalf("read %s without refusing it", c.name)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal says %q, and it has to say %q so the next reader knows what is wrong", err, c.says)
			}
		})
	}
}

func TestLoadNamesTheFileWhenTheCopyIsNotThere(t *testing.T) {
	_, err := Load(t.TempDir())
	if err == nil {
		t.Fatal("loaded a copy that is not in the tree")
	}
	if !strings.Contains(err.Error(), File) {
		t.Errorf("the failure is %q and does not name %s", err, File)
	}
}

// The three shapes a difference comes in. A value that moved is the expected
// one; a value that only one side carries is the one that would go unreported
// if presence were read off an empty string.
func TestCompareNamesAValueThatMovedAndOneThatOnlyOneSideCarries(t *testing.T) {
	pinned := Values{"shape.radius.value": "12", "focus.accent": "#5B9CFF", "surface.line.alpha": "0.07"}
	published := Values{"shape.radius.value": "14", "focus.accent": "#5B9CFF", "type.display.size": "56"}

	got := Compare(pinned, published)
	if len(got) != 3 {
		t.Fatalf("%d difference(s): %v", len(got), got)
	}
	want := []Difference{
		{Name: "shape.radius.value", Pinned: "12", HasPinned: true, Published: "14", HasPublished: true},
		{Name: "surface.line.alpha", Pinned: "0.07", HasPinned: true, HasPublished: false},
		{Name: "type.display.size", HasPinned: false, Published: "56", HasPublished: true},
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("difference %d is %+v, want %+v", i, got[i], w)
		}
	}
}

// Whitespace is not a difference. What this repository vendors is the values,
// and a run reddening because somebody upstream re-indented a file is a run
// people learn to ignore.
func TestRunPassesWhenTheValuesAgreeHoweverThePublishedFileIsFormatted(t *testing.T) {
	root := pin(t, oneGroup)
	var log strings.Builder

	body := "{\n  \"surface\": {\n    \"tokens\": {\n      \"ground\": {\n" +
		"        \"css-var\": \"--ground\",\n        \"dark\": { \"srgb\": \"#121216\", \"alpha\": 1 }\n" +
		"      }\n    }\n  }\n}\n"
	if err := Run(root, answer(body), &log); err != nil {
		t.Fatalf("the run refused a copy whose values agree: %v\n%s", err, log.String())
	}
	if !strings.Contains(log.String(), "3 value(s) pinned, 3 published, 0 differing.") {
		t.Errorf("the run does not say what it compared:\n%s", log.String())
	}
}

// The done-when asks that an introduced difference reds the run and that the
// output names the differing tokens rather than saying only that the copy is
// stale.
func TestRunRedsOnAnIntroducedDifferenceAndNamesTheValues(t *testing.T) {
	root := pin(t, oneGroup)
	var log strings.Builder

	err := Run(root, answer(`{"surface":{"tokens":{"ground":{"css-var":"--ground",
		"dark":{"srgb":"#0B0B0F","alpha":1},"light":{"srgb":"#FAFAFA","alpha":1}}}}}`), &log)
	if err == nil {
		t.Fatalf("the run passed over a copy that differs:\n%s", log.String())
	}
	for _, want := range []string{
		`surface.tokens.ground.dark.srgb: BEHIND, this tree says "#121216", the published file says "#0B0B0F"`,
		`surface.tokens.ground.light.alpha: BEHIND, this tree carries no such value, the published file says "1"`,
		"3 value(s) pinned, 5 published, 3 differing.",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q:\n%s", want, log.String())
		}
	}
}

// A comparison that could not fetch is not a comparison. This is the leg that
// separates a copy nobody checked from a copy somebody checked and found
// current, and collapsing the two is what makes a stale copy invisible.
func TestRunRedsWhenThePublishedFileCouldNotBeRead(t *testing.T) {
	root := pin(t, oneGroup)

	// Each case names what only that arm can say. Both end in the same
	// verdict, so a run reporting a fetch that never happened as a file that
	// would not parse is a message that sends the next reader to the wrong
	// half of the problem, and asserting the verdict alone would not see it.
	for _, c := range []struct {
		name, says string
		fetch      Fetcher
	}{
		{"the fetch itself failed", "dial tcp: no route to host", fails(errors.New("dial tcp: no route to host"))},
		{"what came back is not a token file", "is not JSON", answer("<html>a proxy sign-in page</html>")},
	} {
		t.Run(c.name, func(t *testing.T) {
			var log strings.Builder
			err := Run(root, c.fetch, &log)
			if err == nil {
				t.Fatalf("the run passed having compared nothing:\n%s", log.String())
			}
			if !strings.Contains(log.String(), "UNRESOLVED") {
				t.Errorf("the run does not say the comparison did not happen:\n%s", log.String())
			}
			if !strings.Contains(log.String(), c.says) {
				t.Errorf("the run does not say %q, so it does not say why the comparison did not happen:\n%s", c.says, log.String())
			}
			if !strings.Contains(err.Error(), "not compared") {
				t.Errorf("the failure is %q, and it has to say the copy was not compared", err)
			}
		})
	}
}

// An absent published file is not a failure to fetch and it is not agreement.
// The done-when asks for exactly this state: the run says it compared nothing
// and what would end that, and it does not pass.
func TestRunSaysItComparedNothingWhenNoFileIsPublished(t *testing.T) {
	root := pin(t, oneGroup)
	var log strings.Builder

	err := Run(root, fails(ErrNotPublished), &log)
	if err == nil {
		t.Fatalf("the run passed with nothing on the other side of the comparison:\n%s", log.String())
	}
	for _, want := range []string{"compared nothing", "what ends this state is that file being published"} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("the run does not say %q:\n%s", want, log.String())
		}
	}
	if strings.Contains(log.String(), "BEHIND") {
		t.Errorf("the run reported a difference against a file that is not there:\n%s", log.String())
	}
}

// The run reports and does not write. A run that pulled the published values
// into the tree would destroy the difference, and the difference is the whole
// output.
func TestRunWritesNothingWhateverItFinds(t *testing.T) {
	for _, c := range []struct {
		name  string
		fetch Fetcher
	}{
		{"the values agree", answer(oneGroup)},
		{"the values differ", answer(`{"surface":{"tokens":{"ground":{"css-var":"--x","dark":{"srgb":"#000000","alpha":1}}}}}`)},
		{"nothing is published", fails(ErrNotPublished)},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := pin(t, oneGroup)
			before := walk(t, root)

			var log strings.Builder
			_ = Run(root, c.fetch, &log)

			after := walk(t, root)
			if len(before) != len(after) {
				t.Fatalf("the run changed what is in the tree: %v became %v", before, after)
			}
			for name, body := range before {
				if after[name] != body {
					t.Errorf("the run rewrote %s", name)
				}
			}
		})
	}
}

// The copy this repository actually carries. What is asked is that it parses
// and carries values, rather than what any one of them says: a case asserting a
// colour would be a statement about the day it was written, and the published
// file is the authority for the values.
func TestThePinnedCopyInThisTreeIsAFileThisPackageCanRead(t *testing.T) {
	values, err := Load(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("the copy in this tree does not read: %v", err)
	}
	if len(values) == 0 {
		t.Fatal("the copy in this tree carries no value")
	}
}

func walk(t *testing.T, root string) map[string]string {
	t.Helper()
	found := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(filepath.Clean(p))
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		found[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	return found
}

// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over what the pages say about which server generation a build is
// for.
//
// The fact is one field of one record and it reaches a reader in two places,
// which is what these cases are about. A plugin's own page carries a line per
// generation, because a plugin publishing a build per generation under one
// identity is the case a single line renders wrong while looking exactly as
// correct as a right one. The install page carries the same value as a phrase
// beside the plugin, because that is the reader deciding what to paste into a
// server. Both come out of one read of one field, so they cannot name different
// servers.
//
// The other half is silence. A plugin with no finished release says there is
// nothing to install, and a version printed beside that sentence would be read
// as something to install. Only a published release says which generation a
// build is for, so a plugin that has published none has nothing to say here and
// says nothing.
package site

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Flowfin/site/internal/releases"
)

// generationTemplate renders the paragraphs and the installable list, which is
// where the two spellings of the value land.
const generationTemplate = `<title>{{ .Title }}</title>
<meta name="description" content="{{ .Description }}" />
{{- range .Paragraphs }}
<p>{{ . }}</p>
{{- end }}
{{- if .InstallableRead }}
<p id="installable">{{ .InstallableRead }}</p>
{{- range .Installable }}
<li>{{ .ID }} {{ .Summary }}{{ if .Targets }} Published for {{ .Targets }}.{{ end }}</li>
{{- end }}
{{- end }}
{{- range .Onward }}
<a href="{{ .Href }}">{{ .Text }}</a>
{{- end }}
`

// Three rows: one with a build on two generations, one with a build on one, and
// one with nothing finished published. The two-generation row is the case a
// single-value field renders wrong, and the last is the case that has to stay
// silent.
const (
	threeRows = `[
  {"id":"two","repository":"Flowfin/jellyfin-plugin-two","summary":"What two does","state":"build-up"},
  {"id":"one","repository":"Flowfin/jellyfin-plugin-one","summary":"What one does","state":"shell"},
  {"id":"none","repository":"Flowfin/jellyfin-plugin-none","summary":"What none does","state":"build-up"}
]`
	threeRecorded = `{"taken":"2026-01-02","command":"a command","repositories":{
  "Flowfin/jellyfin-plugin-two":{"finished":2,"prereleases":0,"generations":["10.11","12.0"]},
  "Flowfin/jellyfin-plugin-one":{"finished":1,"prereleases":0,"generations":["10.11"]},
  "Flowfin/jellyfin-plugin-none":{"finished":0,"prereleases":5}}}`
)

// generationTree writes a tree that produces the plugin pages and the install
// page from the rows and the record it is handed.
func generationTree(t *testing.T, roster, record string) string {
	t.Helper()
	root := t.TempDir()
	mkdir(t, filepath.Join(root, TemplatesDir))
	write(t, filepath.Join(root, TemplatesDir, "page.html.tmpl"), generationTemplate)
	mkdir(t, filepath.Join(root, ContentDir))
	write(t, filepath.Join(root, ContentDir, "index.txt"), headIndex)
	write(t, filepath.Join(root, filepath.FromSlash(InstallProse)), installProse)
	mkdir(t, filepath.Join(root, filepath.Dir(filepath.FromSlash(InstallFile))))
	write(t, filepath.Join(root, filepath.FromSlash(InstallFile)), addressAnswered)
	write(t, filepath.Join(root, filepath.FromSlash(RosterFile)), roster)
	write(t, filepath.Join(root, filepath.FromSlash(releases.File)), record)
	proseForEveryRow(t, root, roster)
	return root
}

// pluginPage answers with the produced page for one identifier.
func pluginPage(t *testing.T, root, id string) string {
	t.Helper()
	return read(t, filepath.Join(root, OutputDir, PluginsDir, id, indexDocument))
}

// A plugin whose finished releases span two generations gets a line for each
// rather than one line that is right about half of it. Counted rather than
// contained, because a page carrying both sentences inside one paragraph passes
// every containment check and is the single line this case is against.
func TestTwoGenerationsProduceTwoLines(t *testing.T) {
	root := generationTree(t, threeRows, threeRecorded)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused: %v", err)
	}

	page := pluginPage(t, root, "two")
	if got := strings.Count(page, "<p>A finished release is published for Jellyfin"); got != 2 {
		t.Fatalf("the page carries %d generation line(s) for a plugin published on two:\n%s", got, page)
	}
	for _, want := range []string{
		"<p>A finished release is published for Jellyfin 10.11.</p>",
		"<p>A finished release is published for Jellyfin 12.0.</p>",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page does not carry %q:\n%s", want, page)
		}
	}

	one := pluginPage(t, root, "one")
	if got := strings.Count(one, "<p>A finished release is published for Jellyfin"); got != 1 {
		t.Fatalf("the page carries %d generation line(s) for a plugin published on one:\n%s", got, one)
	}
}

// A plugin with no finished release prints that there is nothing to install and
// no version at all. The prerelease count is deliberately not zero here: a
// prerelease is the one publication such a plugin can have, and a generation
// read out of one would name a server for a build the same page says is not the
// finished one.
func TestAPluginWithNoFinishedReleasePrintsNoVersion(t *testing.T) {
	root := generationTree(t, threeRows, threeRecorded)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused: %v", err)
	}

	page := pluginPage(t, root, "none")
	if strings.Contains(page, "A finished release is published for Jellyfin") {
		t.Errorf("a plugin with no finished release names a server generation:\n%s", page)
	}
	if strings.Contains(page, "10.11") || strings.Contains(page, "12.0") {
		t.Errorf("a plugin with no finished release carries a version:\n%s", page)
	}
	if !strings.Contains(page, "there is nothing to install") {
		t.Errorf("a plugin with no finished release does not say there is nothing to install:\n%s", page)
	}
}

// The install page names the same generations, out of the same field of the same
// record. A second read anywhere would be a second thing to keep current, and
// the reader this fact exists for is the one on that page.
func TestTheInstallPagePrintsTheSameGenerations(t *testing.T) {
	root := generationTree(t, threeRows, threeRecorded)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused: %v", err)
	}

	page := read(t, filepath.Join(root, OutputDir, filepath.FromSlash(InstallPath)))
	if !strings.Contains(page, "<li>two What two does Published for Jellyfin 10.11 and 12.0.</li>") {
		t.Errorf("the install page does not name both generations for the plugin published on two:\n%s", page)
	}
	if !strings.Contains(page, "<li>one What one does Published for Jellyfin 10.11.</li>") {
		t.Errorf("the install page does not name the generation for the plugin published on one:\n%s", page)
	}
	if strings.Contains(page, "<li>none") {
		t.Errorf("the install page offers a plugin with no finished release:\n%s", page)
	}
}

// Moving the generation in the record moves it on both pages. That is what makes
// the value a fact about what was published rather than a sentence somebody
// wrote, and it is the property the whole field exists for.
func TestMovingTheRecordedGenerationMovesBothPages(t *testing.T) {
	root := generationTree(t, threeRows, threeRecorded)
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused: %v", err)
	}
	write(t, filepath.Join(root, filepath.FromSlash(releases.File)),
		strings.Replace(threeRecorded, `"generations":["10.11"]`, `"generations":["12.0"]`, 1))
	if _, err := Build(root, OutputDir, io.Discard); err != nil {
		t.Fatalf("the build refused after the record moved: %v", err)
	}

	page := pluginPage(t, root, "one")
	if strings.Contains(page, "Jellyfin 10.11") {
		t.Errorf("the plugin page still names the generation the record no longer carries:\n%s", page)
	}
	if !strings.Contains(page, "<p>A finished release is published for Jellyfin 12.0.</p>") {
		t.Errorf("the plugin page does not name the generation the record now carries:\n%s", page)
	}
	install := read(t, filepath.Join(root, OutputDir, filepath.FromSlash(InstallPath)))
	if !strings.Contains(install, "<li>one What one does Published for Jellyfin 12.0.</li>") {
		t.Errorf("the install page did not follow the record:\n%s", install)
	}
}

// A record claiming a plugin ships and saying nothing about which server it is
// for reds the build rather than producing a page a reader would act on. The
// refusal is the record's, and this case is that the build carries it rather
// than rendering past it.
func TestABuildOverARecordWithNoGenerationIsRefused(t *testing.T) {
	root := generationTree(t, threeRows,
		strings.Replace(threeRecorded, `"finished":1,"prereleases":0,"generations":["10.11"]`, `"finished":1,"prereleases":0`, 1))
	_, err := Build(root, OutputDir, io.Discard)
	if err == nil {
		t.Fatal("the build produced pages from a record that says a plugin ships and not which server for")
	}
	if !strings.Contains(err.Error(), "no server generation") {
		t.Fatalf("the refusal does not say what was missing: %v", err)
	}
}

// The third state on a page: finished releases that publish nothing about which
// server they are for. A page listing the generations some of the releases state
// and saying nothing about the rest reads as a page that read all of them, which
// is the failure the count exists against.
func TestReleasesStatingNoGenerationAreSaidRatherThanDropped(t *testing.T) {
	for name, tc := range map[string]struct {
		row  plugin
		says string
	}{
		"one further release states none": {
			plugin{Generations: []string{"10.11"}, Unstated: 1},
			"One further finished release publishes nothing about which server generation it is for"},
		"several further releases state none": {
			plugin{Generations: []string{"10.11", "12.0"}, Unstated: 4},
			"A further 4 finished releases publish nothing"},
		"the only release states none": {
			plugin{Unstated: 1},
			"The finished release publishes nothing about which server generation it is for"},
		"every release states none": {
			plugin{Unstated: 3},
			"All 3 finished releases publish nothing"},
	} {
		lines := strings.Join(generationLines(tc.row), "\n")
		if !strings.Contains(lines, tc.says) {
			t.Errorf("%s: the lines do not say %q:\n%s", name, tc.says, lines)
		}
	}
}

// A plugin whose every finished release states no generation names none rather
// than one nothing published says, and it is the case a reading that took the
// first generation it found would get wrong by having none to find.
func TestWhereNoReleaseStatesAGenerationThePageNamesNone(t *testing.T) {
	lines := strings.Join(generationLines(plugin{Unstated: 2}), " ")
	if strings.Contains(lines, "published for Jellyfin") {
		t.Errorf("a plugin whose releases state no generation names one:\n%s", lines)
	}
	if !strings.Contains(lines, "All 2 finished releases publish nothing") {
		t.Errorf("the page does not say that nothing published states a generation:\n%s", lines)
	}
}

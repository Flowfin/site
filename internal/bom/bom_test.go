// The suite over the bill of materials.
//
// Two questions, and they are different. Does the document list what goes into
// producing the published bytes, and does it stop rather than describe a tree it
// could not read. A document that quietly omitted the base image would read
// exactly like one produced from a tree that has none, which is the failure the
// second half exists for.
package bom

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Flowfin/site/internal/pins"
)

const (
	sampleGoMod = `module github.com/Flowfin/site

go 1.26.0

toolchain go1.26.5
`
	sampleDockerfile = `FROM golang:1.26.5-bookworm@sha256:6c5605ab3a9a9fb3c4eafe5b3d63cdbf3881caf113262b67862547b54a9db599

WORKDIR /src
`
	samplePins = `[
  {
    "id": "prettier",
    "registry": "npm",
    "name": "prettier",
    "version": "3.9.6",
    "checksum": "",
    "reads": [".github/workflows/prettier.yml"],
    "repeats": [],
    "why": "The formatter is fetched at run time and no updater sees it."
  }
]
`
	pinsWithoutTheFormatter = `[
  {
    "id": "zizmor",
    "registry": "pypi",
    "name": "zizmor",
    "version": "1.26.1",
    "checksum": "",
    "reads": [".github/workflows/zizmor.yml"],
    "repeats": [],
    "why": "The workflow audit is fetched at run time and no updater sees it."
  }
]
`
)

// tree writes the three files the document is derived from, so a test changes one
// of them and reads what the document did about it.
func tree(t *testing.T, gomod, dockerfile, pinned string) string {
	t.Helper()

	root := t.TempDir()
	write := func(name, body string) {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("preparing %s: %v", name, err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	if gomod != "" {
		write(GoModFile, gomod)
	}
	if dockerfile != "" {
		write(DockerfileTxt, dockerfile)
	}
	if pinned != "" {
		write(pins.File, pinned)
	}
	return root
}

// The three things the document owes, each read off the document rather than off
// the file it came from.
func TestTheDocumentCarriesTheToolchainTheGraphAndTheBaseImage(t *testing.T) {
	doc, err := Build(tree(t, sampleGoMod, sampleDockerfile, samplePins))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if doc.Format != "CycloneDX" || doc.SpecVersion != SpecVersion {
		t.Errorf("the document declares %s %s, which is not the format it says it is", doc.Format, doc.SpecVersion)
	}

	byName := map[string]Component{}
	for _, c := range doc.Components {
		byName[c.Name] = c
	}

	toolchain, ok := byName["go"]
	if !ok {
		t.Fatalf("the document lists no toolchain; it lists %v", Names(doc))
	}
	if toolchain.Version != "1.26.5" {
		t.Errorf("the toolchain reads %q, and go.mod pins 1.26.5", toolchain.Version)
	}

	image, ok := byName["golang"]
	if !ok {
		t.Fatalf("the document lists no base image; it lists %v", Names(doc))
	}
	if len(image.Hashes) != 1 || image.Hashes[0].Content != "6c5605ab3a9a9fb3c4eafe5b3d63cdbf3881caf113262b67862547b54a9db599" {
		t.Errorf("the base image carries %v rather than the digest the Dockerfile pins", image.Hashes)
	}

	formatter, ok := byName["prettier"]
	if !ok {
		t.Fatalf("the document lists no formatter; it lists %v", Names(doc))
	}
	if formatter.Version != "3.9.6" {
		t.Errorf("the formatter reads %q, and the pins file declares 3.9.6", formatter.Version)
	}

	// The graph is empty in this fixture, and an empty graph is a fact the
	// document states by listing no library rather than by listing nothing.
	for _, c := range doc.Components {
		if c.Type == "library" {
			t.Errorf("the document lists %s as a library, and this fixture requires no module", c.Name)
		}
	}
}

// The line the issue closes on. A module added to the tree is a module in the
// document produced from that tree, so the change and the entry cannot arrive in
// different pull requests.
func TestAModuleAddedToTheTreeIsInTheDocument(t *testing.T) {
	before, err := Build(tree(t, sampleGoMod, sampleDockerfile, samplePins))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	withModule := sampleGoMod + `
require (
	example.com/one v1.2.3
	example.com/two v0.4.0 // indirect
)
`
	after, err := Build(tree(t, withModule, sampleDockerfile, samplePins))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(after.Components) != len(before.Components)+2 {
		t.Fatalf("the document listed %d component(s) before and %d after two modules were added",
			len(before.Components), len(after.Components))
	}
	listed := strings.Join(Names(after), "\n")
	for _, want := range []string{"example.com/one v1.2.3", "example.com/two v0.4.0"} {
		if !strings.Contains(listed, want) {
			t.Errorf("the document does not list %q; it lists %v", want, Names(after))
		}
	}
	for _, c := range after.Components {
		if c.Name == "example.com/one" && c.PURL != "pkg:golang/example.com/one@v1.2.3" {
			t.Errorf("the module carries %q, which is not a package address a reader can resolve", c.PURL)
		}
	}
}

// A single-line require is the other spelling and it is the one a tree grows
// first, so the document reads it too.
func TestASingleRequireLineIsRead(t *testing.T) {
	doc, err := Build(tree(t, sampleGoMod+"\nrequire example.com/one v1.2.3\n", sampleDockerfile, samplePins))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(strings.Join(Names(doc), "\n"), "example.com/one v1.2.3") {
		t.Errorf("the document does not list a module required on one line; it lists %v", Names(doc))
	}
}

// Each source refused in turn. A document produced without one of them would say
// the same thing as a document produced from a tree that has none, and the whole
// value of this file is that those two are not the same statement.
func TestASourceThatCannotBeReadStopsTheDocument(t *testing.T) {
	for name, roots := range map[string]struct{ gomod, dockerfile, pinned, want string }{
		"no go.mod":     {"", sampleDockerfile, samplePins, GoModFile},
		"no Dockerfile": {sampleGoMod, "", samplePins, DockerfileTxt},
		"no pins file":  {sampleGoMod, sampleDockerfile, "", pins.File},
		"a base image pinned by tag alone": {sampleGoMod,
			"FROM golang:1.26.5-bookworm\n", samplePins, DockerfileTxt},
		"a pins file declaring no formatter": {sampleGoMod, sampleDockerfile,
			pinsWithoutTheFormatter, pins.File},
		"a go.mod naming no toolchain": {"module github.com/Flowfin/site\n",
			sampleDockerfile, samplePins, GoModFile},
	} {
		_, err := Build(tree(t, roots.gomod, roots.dockerfile, roots.pinned))
		if err == nil {
			t.Errorf("%s produced a document rather than a failure", name)
			continue
		}
		if !strings.Contains(err.Error(), roots.want) {
			t.Errorf("%s failed with %q, which does not name %s", name, err, roots.want)
		}
	}
}

// Two runs over one source produce one document. A timestamp or a serial number
// would make a bill of materials something nobody can compare between runs, which
// is most of what it is for.
func TestTwoRunsOverOneSourceProduceTheSameBytes(t *testing.T) {
	root := tree(t, sampleGoMod, sampleDockerfile, samplePins)

	var first, second bytes.Buffer
	if err := Write(root, &first, io.Discard); err != nil {
		t.Fatalf("the first run: %v", err)
	}
	if err := Write(root, &second, io.Discard); err != nil {
		t.Fatalf("the second run: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Error("two runs over one source produced different documents")
	}

	var parsed map[string]any
	if err := json.Unmarshal(first.Bytes(), &parsed); err != nil {
		t.Fatalf("the document is not JSON: %v", err)
	}
	for _, unwanted := range []string{"timestamp", "serialNumber"} {
		if _, ok := parsed[unwanted]; ok {
			t.Errorf("the document carries %s, which moves between two runs of one source", unwanted)
		}
	}
}

// The run says what it covered. A document redirected into a file leaves the
// reader with this and nothing else, so a run that listed one component cannot be
// read as one that listed the tree.
func TestTheRunReportsWhatItCovered(t *testing.T) {
	var doc, report bytes.Buffer
	if err := Write(tree(t, sampleGoMod, sampleDockerfile, samplePins), &doc, &report); err != nil {
		t.Fatalf("Write: %v", err)
	}
	for _, want := range []string{"CycloneDX " + SpecVersion, "3 component(s)", "the module graph is empty"} {
		if !strings.Contains(report.String(), want) {
			t.Errorf("the report does not say %q; it said:\n%s", want, report.String())
		}
	}
}

// The document produced from this repository, which is the one the run attaches.
// A suite that only ever judged a fixture would not notice the day a real source
// file changed shape.
func TestThisRepositoryProducesADocument(t *testing.T) {
	doc, err := Build("../..")
	if err != nil {
		t.Fatalf("the document for this repository: %v", err)
	}
	listed := strings.Join(Names(doc), "\n")
	for _, want := range []string{"go 1.26.5", "golang 1.26.5-bookworm", "prettier"} {
		if !strings.Contains(listed, want) {
			t.Errorf("the document for this repository does not list %q; it lists %v", want, Names(doc))
		}
	}
}

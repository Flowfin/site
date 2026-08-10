// Package bom writes the bill of materials for what goes into producing the
// published bytes.
//
// This repository intends to carry no application dependency, and that intention
// is worth what the document proving it is worth. The interesting case is the day
// the number stops being zero, so the document is produced from the tree rather
// than written by hand, and a source it cannot read is a failure rather than an
// omission: a document that quietly left out the base image would read exactly
// like one produced from a tree that has none.
//
// Four sources, and each one is the file that decides the thing rather than a
// second copy of it. The toolchain and the module graph come from go.mod, the
// base image from the Dockerfile, and the formatter version from the file that
// declares the pins no updater watches. The workflow that fetches the formatter
// is named in what the document says about it and is not read for the number,
// because the number is no longer written there.
//
// Nothing here reads the clock or the network. CycloneDX allows a timestamp and a
// serial number in the metadata and both are left out on purpose: they would make
// two runs over one source produce different documents, which is the property the
// reproducibility check exists to hold, and a bill of materials that cannot be
// compared between two runs is a document nobody can check.
package bom

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Flowfin/site/internal/pins"
)

// The files the document is derived from. They are named here rather than passed
// in, because which file decides the base image is a fact about this repository
// and not a parameter of a run.
const (
	GoModFile     = "go.mod"
	DockerfileTxt = "Dockerfile"
	FormatterFile = ".github/workflows/prettier.yml"
)

// FormatterPin is the identifier the formatter is declared under in the pins
// file. The document names the pin rather than searching the file for something
// that looks like a formatter, so a renamed pin is a failure here instead of a
// component quietly dropping out of the document.
const FormatterPin = "prettier"

// SpecVersion is the CycloneDX version the document declares. A standard format
// rather than one invented here, so a reader can put the document through a tool
// they already have.
const SpecVersion = "1.6"

// Document is the bill of materials, in the subset of CycloneDX this repository
// can fill honestly. A field nothing in the tree decides is absent rather than
// present and empty.
type Document struct {
	Format      string      `json:"bomFormat"`
	SpecVersion string      `json:"specVersion"`
	Version     int         `json:"version"`
	Metadata    Metadata    `json:"metadata"`
	Components  []Component `json:"components"`
}

// Metadata carries the thing the document is about.
type Metadata struct {
	Component Component `json:"component"`
}

// Component is one thing that goes into producing the published bytes.
type Component struct {
	Type        string `json:"type"`
	BOMRef      string `json:"bom-ref"`
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	PURL        string `json:"purl,omitempty"`
	Description string `json:"description,omitempty"`
	Hashes      []Hash `json:"hashes,omitempty"`
}

// Hash is a digest a reader can compare against what they pulled.
type Hash struct {
	Algorithm string `json:"alg"`
	Content   string `json:"content"`
}

var (
	moduleLine    = regexp.MustCompile(`(?m)^module\s+(\S+)\s*$`)
	toolchainLine = regexp.MustCompile(`(?m)^toolchain\s+go(\S+)\s*$`)
	goLine        = regexp.MustCompile(`(?m)^go\s+(\S+)\s*$`)
	requireOne    = regexp.MustCompile(`(?m)^require\s+(\S+)\s+(\S+)\s*(//.*)?$`)
	requireBlock  = regexp.MustCompile(`(?sm)^require\s*\((.*?)\n\)`)
	requireEntry  = regexp.MustCompile(`(?m)^\s*(\S+)\s+(v\S+)\s*(//.*)?$`)
	fromLine      = regexp.MustCompile(`(?m)^FROM\s+([^\s@:]+)(?::(\S+))?@sha256:([0-9a-f]{64})\s*$`)
)

// Build reads the tree at root and returns the document. Every source has to
// answer; the first one that does not stops the run and says which file it was.
func Build(root string) (Document, error) {
	gomod, err := read(root, GoModFile)
	if err != nil {
		return Document{}, err
	}

	module := moduleLine.FindSubmatch(gomod)
	if module == nil {
		return Document{}, fmt.Errorf("%s declares no module path, so there is nothing for the document to be about", GoModFile)
	}
	path := string(module[1])

	toolchain, err := toolchainOf(gomod)
	if err != nil {
		return Document{}, err
	}

	components := []Component{{
		Type:        "platform",
		BOMRef:      "toolchain/go@" + toolchain,
		Name:        "go",
		Version:     toolchain,
		PURL:        "pkg:golang/go@" + toolchain,
		Description: "the toolchain " + GoModFile + " pins, which is the one the build refuses to run without",
	}}

	for _, m := range modules(gomod) {
		components = append(components, m)
	}

	image, err := baseImage(root)
	if err != nil {
		return Document{}, err
	}
	components = append(components, image)

	formatter, err := formatterVersion(root)
	if err != nil {
		return Document{}, err
	}
	components = append(components, formatter)

	return Document{
		Format:      "CycloneDX",
		SpecVersion: SpecVersion,
		Version:     1,
		Metadata: Metadata{Component: Component{
			Type:        "application",
			BOMRef:      path,
			Name:        path,
			PURL:        "pkg:golang/" + path,
			Description: "the generator that renders this site, and the pages it produces",
		}},
		Components: components,
	}, nil
}

// Write renders the document to w, and reports what it covered so a run that
// listed nothing cannot be read as a tree that needed nothing listed.
func Write(root string, out, log io.Writer) error {
	doc, err := Build(root)
	if err != nil {
		return err
	}

	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if _, err := out.Write(append(body, '\n')); err != nil {
		return err
	}

	modules := 0
	for _, c := range doc.Components {
		if c.Type == "library" {
			modules++
		}
	}
	fmt.Fprintf(log, "sbom: CycloneDX %s, %d component(s) for %s\n", doc.SpecVersion, len(doc.Components), doc.Metadata.Component.Name)
	fmt.Fprintf(log, "  read %s, %s and %s\n", GoModFile, DockerfileTxt, pins.File)
	if modules == 0 {
		fmt.Fprintf(log, "  the module graph is empty, so the document lists the toolchain, the base image and the formatter and no library\n")
	} else {
		fmt.Fprintf(log, "  %d module(s) in the graph %s declares\n", modules, GoModFile)
	}
	return nil
}

// toolchainOf prefers the pinned toolchain over the language version, because the
// pin is what actually compiled the bytes. A tree with neither is refused rather
// than described by a guess.
func toolchainOf(gomod []byte) (string, error) {
	if m := toolchainLine.FindSubmatch(gomod); m != nil {
		return string(m[1]), nil
	}
	if m := goLine.FindSubmatch(gomod); m != nil {
		return string(m[1]), nil
	}
	return "", fmt.Errorf("%s names neither a toolchain nor a language version, so what compiled the bytes cannot be stated", GoModFile)
}

// modules reads what go.mod requires. That is the graph a tidied module builds
// against, and it is read from the file rather than from `go list -m all`,
// because the suite that proves this runs on a clean clone with no network and a
// command that resolves the graph does not.
func modules(gomod []byte) []Component {
	var found []Component
	add := func(path, version string) {
		found = append(found, Component{
			Type:    "library",
			BOMRef:  path + "@" + version,
			Name:    path,
			Version: version,
			PURL:    "pkg:golang/" + path + "@" + version,
		})
	}

	if m := requireOne.FindSubmatch(gomod); m != nil {
		add(string(m[1]), string(m[2]))
	}
	for _, block := range requireBlock.FindAllSubmatch(gomod, -1) {
		for _, e := range requireEntry.FindAllSubmatch(block[1], -1) {
			add(string(e[1]), string(e[2]))
		}
	}
	return found
}

// baseImage reads the digest the container is pinned to. A tag on its own is not
// enough: the digest is what says which bytes were pulled, and a document naming
// only the tag describes whatever that name points at today.
func baseImage(root string) (Component, error) {
	body, err := read(root, DockerfileTxt)
	if err != nil {
		return Component{}, err
	}
	m := fromLine.FindSubmatch(body)
	if m == nil {
		return Component{}, fmt.Errorf("%s carries no base image pinned by digest, and a tag on its own does not say which bytes were pulled", DockerfileTxt)
	}

	name, tag, digest := string(m[1]), string(m[2]), string(m[3])
	version := tag
	if version == "" {
		version = "sha256:" + digest
	}
	return Component{
		Type:        "container",
		BOMRef:      "container/" + name + "@sha256:" + digest,
		Name:        name,
		Version:     version,
		PURL:        "pkg:docker/" + name + "@sha256:" + digest,
		Description: "the base image the container route builds on, pinned by digest in " + DockerfileTxt,
		Hashes:      []Hash{{Algorithm: "SHA-256", Content: digest}},
	}, nil
}

// formatterVersion reads the pin the formatter is fetched at. It is fetched at
// check time and appears in no manifest and no lockfile, which is exactly the
// kind of thing a bill of materials exists to make visible.
func formatterVersion(root string) (Component, error) {
	declared, err := pins.Load(root)
	if err != nil {
		return Component{}, err
	}
	for _, p := range declared {
		if p.ID != FormatterPin {
			continue
		}
		return Component{
			Type:        "application",
			BOMRef:      "formatter/" + p.Name + "@" + p.Version,
			Name:        p.Name,
			Version:     p.Version,
			PURL:        "pkg:npm/" + p.Name + "@" + p.Version,
			Description: "declared in " + pins.File + " and fetched at check time by " + FormatterFile + ", with no manifest and no lockfile in the tree",
		}, nil
	}
	return Component{}, fmt.Errorf("%s declares no pin called %q, so a thing the gate fetches would go unlisted", pins.File, FormatterPin)
}

func read(root, name string) ([]byte, error) {
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}
	return body, nil
}

// Names is what the document lists, in order, for a message that has to say what
// was covered without printing the whole document.
func Names(doc Document) []string {
	var out []string
	for _, c := range doc.Components {
		out = append(out, strings.TrimSpace(c.Name+" "+c.Version))
	}
	return out
}

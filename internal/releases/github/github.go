// SPDX-License-Identifier: AGPL-3.0-or-later

// Package github asks a repository what it has published, and it is the one
// thing under internal/releases that leaves the machine.
//
// It is a package of its own rather than a file, so that what a build reads and
// what a refresh asks are separated by an import rather than by a name. The
// build reads internal/releases and never this, which is a property of the
// dependency graph rather than of anybody remembering it, and the suite over
// this repository is what reads that graph.
package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Flowfin/site/internal/releases"
)

// API is where a release list is published. The path is completed with the
// repository, and the page size is the largest the interface allows so that a
// repository with a long history is read in one request.
const API = "https://api.github.com/repos/"

// Command is what the record records as having produced it. It is the shape of
// the request rather than a shell line, because what the verb makes is an HTTP
// request and a shell command in the record would name a route nobody took.
const Command = "go run . releases, which reads " + API +
	"OWNER/NAME/releases?per_page=100 for each repository the roster names, and the " +
	metadataSuffix + " asset of each finished release for the server generation it targets"

// Fetch reads one repository's release list and counts the two things the
// rule needs.
//
// It counts rather than returning the list, because what the record holds is
// what the rule reads, and a record carrying every tag would be a copy of
// somebody else's data that goes stale in more ways than it is used.
//
// A repository that answers anything but success is refused rather than recorded
// with no releases. The two are opposite statements: a repository with nothing
// published is the normal case for most of the twelve, and a repository that
// could not be read is a question nobody answered.
func Fetch(repository string) (releases.Repository, error) {
	address := API + escapeRepository(repository) + "/releases?per_page=100"

	req, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return releases.Repository{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	authorise(req)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return releases.Repository{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return releases.Repository{}, fmt.Errorf("%s answered %s", address, resp.Status)
	}

	var published []release
	if err := json.NewDecoder(resp.Body).Decode(&published); err != nil {
		return releases.Repository{}, fmt.Errorf("reading what %s answered: %w", address, err)
	}

	r := count(published)
	generations, unstated, err := generationsOf(client.Get, published)
	if err != nil {
		return releases.Repository{}, err
	}
	r.Generations = generations
	r.Unstated = unstated
	return r, nil
}

// release is what this package reads out of one entry of a release list. The
// flags decide which count it lands in, and the assets are where the generation
// it targets is published.
type release struct {
	Tag        string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// metadataSuffix is what the published metadata beside a release archive is
// named. It is the file a Jellyfin server reads to decide whether a build fits
// it, which is why it is the authority here: the same bytes that tell a server
// to refuse a build are what this record says the build is for.
const metadataSuffix = ".meta.json"

// generationsOf reads the generation each finished release targets and answers
// with the distinct ones, newest release first.
//
// Only finished releases are read. decisions/0009-what-counts-as-shipping.md
// keeps a prerelease out of the state the pages compute, and a generation taken
// from one would put a server version on a page beside a sentence saying the
// build it belongs to is not the finished one. That is the reading this record
// takes and it is the reading the pages are written against.
//
// A finished release that publishes no metadata at all is counted rather than
// refused, and the count is returned beside the generations. That is a third
// state and not a silence: a release stating no generation and a run that could
// not read one are different, and the plugin that ships today published four
// finished releases before it published the metadata a generation is read out
// of, so a rule refusing this state could not record that repository at all.
//
// A release that publishes metadata this run could not read or could not parse
// is still refused. That is the run failing rather than the release saying
// nothing, and recording it as the state above would turn a failure into a
// claim about somebody's release.
func generationsOf(get getter, published []release) ([]string, int, error) {
	var out []string
	unstated := 0
	seen := map[string]bool{}
	for _, p := range published {
		if p.Draft || p.Prerelease {
			continue
		}
		g, err := generationOf(get, p)
		switch {
		case errors.Is(err, errNoMetadata):
			unstated++
			continue
		case err != nil:
			return nil, 0, err
		}
		if seen[g] {
			continue
		}
		seen[g] = true
		out = append(out, g)
	}
	return out, unstated, nil
}

// errNoMetadata is the release publishing nothing a generation could be read
// out of. It is a value rather than a message so that the caller can tell it
// from a run that failed, which is the distinction the third state rests on.
var errNoMetadata = errors.New("the release publishes no metadata asset")

// getter is how the metadata beside a release is reached. It is a parameter
// rather than a client so that the reading above can be decided by a case that
// opens no connection: what this package refuses to guess is worth a proof, and
// a proof that needed the network would not be run.
type getter func(address string) (*http.Response, error)

// generationOf reads one release's published metadata and answers with the
// generation it declares, in the spelling a page states rather than the
// four-part number the metadata carries.
func generationOf(get getter, p release) (string, error) {
	var from string
	for _, a := range p.Assets {
		if strings.HasSuffix(a.Name, metadataSuffix) {
			from = a.URL
			break
		}
	}
	if from == "" {
		return "", errNoMetadata
	}

	resp, err := get(from)
	if err != nil {
		return "", fmt.Errorf("reading the metadata of the release %s: %w", p.Tag, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s answered %s", from, resp.Status)
	}

	var meta struct {
		TargetAbi string `json:"targetAbi"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", fmt.Errorf("reading the metadata of the release %s: %w", p.Tag, err)
	}
	g, err := Generation(meta.TargetAbi)
	if err != nil {
		return "", fmt.Errorf("the release %s: %w", p.Tag, err)
	}
	return g, nil
}

// Generation is the server generation a target ABI names, which is its first
// two parts.
//
// The metadata carries four, and every place this project publishes the
// generation in words carries two: the organisation profile, the plugin
// documents and the sentence a server administrator reads. So the record holds
// the two-part form and a page states it rather than deriving anything, and the
// derivation is here, once, with a case behind it.
//
// A value that is not an ABI is refused rather than shortened. An ABI is what a
// server compares a build against, so a value this cannot read is a value
// nothing on a page should be built out of.
func Generation(abi string) (string, error) {
	parts := strings.Split(strings.TrimSpace(abi), ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("declares the target ABI %q, which names no server generation", abi)
	}
	for _, part := range parts[:2] {
		if part == "" || strings.TrimLeft(part, "0123456789") != "" {
			return "", fmt.Errorf("declares the target ABI %q, which is not a number a server compares a build against", abi)
		}
	}
	return parts[0] + "." + parts[1], nil
}

// count applies the reading decisions/0009 fixes: the flag decides and the tag
// string is not read. A draft is not published at all, so it is in neither
// count.
func count(published []release) releases.Repository {
	var r releases.Repository
	for _, p := range published {
		switch {
		case p.Draft:
		case p.Prerelease:
			r.Prereleases++
		default:
			r.Finished++
		}
	}
	return r
}

// escapeRepository escapes each half of `owner/name` and puts the separator
// back, so a value out of the roster cannot add a segment to the address.
func escapeRepository(repository string) string {
	owner, name, found := strings.Cut(repository, "/")
	if !found {
		return url.PathEscape(repository)
	}
	return url.PathEscape(owner) + "/" + url.PathEscape(name)
}

// TokenVariable is the environment variable a token is read from. It is the
// name a workflow already puts a token under, so a run inside one asks with it
// by having it in the environment rather than by being told to.
const TokenVariable = "GITHUB_TOKEN"

// authorise puts a token on the request where the environment holds one.
//
// Without it the request is anonymous, and an anonymous caller is held to a
// rate the twelve repositories here exhaust: a scheduled run from a shared
// address answered 403 for the fifth repository it asked about on 2026-09-01,
// which is a run that says nothing about the record rather than one that read
// it and found it current.
//
// It is optional rather than required, because a contributor asking about
// twelve repositories once is inside the anonymous rate and should not have to
// hold a credential to run a verb. A token that is not there is not an error
// and the run says nothing about it: what it would report is the state of
// somebody's environment, and the refusal that matters is the one the request
// itself answers with.
//
// Only this request carries it. The metadata beside a release is fetched from
// wherever the release list says, which is not this interface and is not a host
// this repository chose, and a credential sent there is a credential handed to
// a third party.
func authorise(req *http.Request) {
	token := strings.TrimSpace(os.Getenv(TokenVariable))
	if token == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
}

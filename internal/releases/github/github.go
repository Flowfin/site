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
	"fmt"
	"net/http"
	"net/url"
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
	"OWNER/NAME/releases?per_page=100 for each repository the roster names"

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

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return releases.Repository{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return releases.Repository{}, fmt.Errorf("%s answered %s", address, resp.Status)
	}

	var published []struct {
		Draft      bool `json:"draft"`
		Prerelease bool `json:"prerelease"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&published); err != nil {
		return releases.Repository{}, fmt.Errorf("reading what %s answered: %w", address, err)
	}

	return count(published), nil
}

// count applies the reading decisions/0009 fixes: the flag decides and the tag
// string is not read. A draft is not published at all, so it is in neither
// count.
func count(published []struct {
	Draft      bool `json:"draft"`
	Prerelease bool `json:"prerelease"`
}) releases.Repository {
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

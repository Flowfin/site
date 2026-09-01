// SPDX-License-Identifier: AGPL-3.0-or-later

// The suite over the one derivation this package makes.
//
// Nothing here reaches the network. What asks a repository anything takes a
// client and an address, and the only thing in this package that decides
// something rather than fetching it is the reading below: what a target ABI
// says about which server generation a build is for.
package github

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// The metadata carries four parts and every place this project publishes the
// generation in words carries two, so the reading is the first two and the rest
// is the build's own version rather than the server's.
func TestTheGenerationIsTheFirstTwoPartsOfTheAbi(t *testing.T) {
	for abi, want := range map[string]string{
		"10.11.0.0":   "10.11",
		"12.0.0.0":    "12.0",
		"10.11":       "10.11",
		"10.11.5":     "10.11",
		" 10.11.0.0 ": "10.11",
	} {
		got, err := Generation(abi)
		if err != nil {
			t.Errorf("%q was refused: %v", abi, err)
			continue
		}
		if got != want {
			t.Errorf("%q read as generation %q, want %q", abi, got, want)
		}
	}
}

// A value that is not an ABI is refused rather than shortened. An ABI is what a
// server compares a build against, so a value this cannot read is one nothing on
// a page should be built out of, and a reading that took the first two parts of
// whatever arrived would put a number in front of a reader deciding what to
// install.
func TestAValueThatIsNotAnAbiIsRefused(t *testing.T) {
	for name, abi := range map[string]string{
		"nothing at all":       "",
		"whitespace":           "   ",
		"one part":             "10",
		"a word":               "latest",
		"a word in the parts":  "ten.eleven.0.0",
		"a part that is empty": "10..0.0",
		"a range":              "10.11-12.0",
	} {
		if got, err := Generation(abi); err == nil {
			t.Errorf("%s (%q) read as the generation %q rather than being refused", name, abi, got)
		}
	}
}

// served answers with the bodies it was given, keyed by address, and refuses an
// address nobody put in it. It opens no connection and binds nothing, which is
// what lets the reading below be decided by the suite at all.
func served(bodies map[string]string) getter {
	return func(address string) (*http.Response, error) {
		body, ok := bodies[address]
		if !ok {
			return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found",
				Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK",
			Body: io.NopCloser(strings.NewReader(body))}, nil
	}
}

// metadata is one release with a metadata asset at the address given.
func metadata(tag, address string) release {
	return release{Tag: tag, Assets: []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	}{{Name: tag + metadataSuffix, URL: address}}}
}

// The distinct generations, newest release first, and a release publishing no
// metadata counted rather than dropped and rather than refused. Both halves
// matter: dropping it would make a run that read some of the releases read like
// one that read all of them, and refusing it could not record the plugin that
// ships today, whose four oldest finished releases publish no metadata at all.
func TestAReleaseStatingNoGenerationIsCountedRatherThanDropped(t *testing.T) {
	published := []release{
		metadata("newest", "https://example.invalid/newest"),
		metadata("older", "https://example.invalid/older"),
		metadata("same-again", "https://example.invalid/same-again"),
		{Tag: "oldest"},
		{Tag: "older-still"},
		{Tag: "a-prerelease", Prerelease: true},
		{Tag: "a-draft", Draft: true},
	}
	get := served(map[string]string{
		"https://example.invalid/newest":     `{"targetAbi":"12.0.0.0"}`,
		"https://example.invalid/older":      `{"targetAbi":"10.11.0.0"}`,
		"https://example.invalid/same-again": `{"targetAbi":"10.11.0.0"}`,
	})

	generations, unstated, err := generationsOf(get, published)
	if err != nil {
		t.Fatalf("the read was refused: %v", err)
	}
	if len(generations) != 2 || generations[0] != "12.0" || generations[1] != "10.11" {
		t.Errorf("the generations read as %q, and they are the distinct ones newest first", generations)
	}
	if unstated != 2 {
		t.Errorf("%d release(s) counted as stating no generation, and two of the finished ones publish none", unstated)
	}
}

// A prerelease is read for nothing. decisions/0009-what-counts-as-shipping.md
// keeps it out of the state the pages compute, and a generation taken from one
// would name a server for a build the same page says is not the finished one.
func TestAPrereleaseIsNotReadForItsGeneration(t *testing.T) {
	asked := map[string]bool{}
	get := func(address string) (*http.Response, error) {
		asked[address] = true
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK",
			Body: io.NopCloser(strings.NewReader(`{"targetAbi":"12.0.0.0"}`))}, nil
	}
	published := []release{
		metadata("finished", "https://example.invalid/finished"),
		{Tag: "prerelease", Prerelease: true, Assets: metadata("prerelease", "https://example.invalid/prerelease").Assets},
	}
	generations, unstated, err := generationsOf(get, published)
	if err != nil {
		t.Fatalf("the read was refused: %v", err)
	}
	if asked["https://example.invalid/prerelease"] {
		t.Error("the prerelease was read for a generation")
	}
	if len(generations) != 1 || unstated != 0 {
		t.Errorf("the read answered %q and %d stating none", generations, unstated)
	}
}

// A release that publishes metadata this run could not read is the run failing
// rather than the release saying nothing, and recording it as the third state
// would turn a failure into a claim about somebody's release.
func TestMetadataThatCouldNotBeReadIsAFailureRatherThanTheThirdState(t *testing.T) {
	for name, get := range map[string]getter{
		"the address does not answer": served(nil),
		"the metadata is not the object it has to be": served(map[string]string{
			"https://example.invalid/one": "not an object"}),
		"the metadata states no target": served(map[string]string{
			"https://example.invalid/one": `{}`}),
		"the metadata states a target that is not one": served(map[string]string{
			"https://example.invalid/one": `{"targetAbi":"latest"}`}),
	} {
		_, unstated, err := generationsOf(get, []release{metadata("one", "https://example.invalid/one")})
		if err == nil {
			t.Errorf("%s: the read answered with %d stating none rather than failing", name, unstated)
		}
		if err != nil && errors.Is(err, errNoMetadata) {
			t.Errorf("%s: the failure was reported as the release publishing nothing", name)
		}
	}
}

// The token reaches the request that asks this interface and nothing else. A
// credential put on the request for the metadata beside a release would be
// handed to whichever host that release names, which is not this interface and
// is not a host this repository chose.
func TestTheTokenReachesTheReleaseListAndNothingElse(t *testing.T) {
	t.Setenv(TokenVariable, "  a-token  ")

	req, err := http.NewRequest(http.MethodGet, API+"a/b/releases?per_page=100", nil)
	if err != nil {
		t.Fatalf("building the request: %v", err)
	}
	authorise(req)
	if got := req.Header.Get("Authorization"); got != "Bearer a-token" {
		t.Errorf("the request carries the authorisation %q", got)
	}

	// The metadata is reached through the getter below, and a case that
	// sees a header on it is a case that has caught the leak.
	asked := ""
	_, _, err = generationsOf(func(address string) (*http.Response, error) {
		asked = address
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(`{"targetAbi":"10.11.0.0"}`)),
		}, nil
	}, []release{{Tag: "1.0", Assets: []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	}{{Name: "a" + metadataSuffix, URL: "https://elsewhere.example/a.meta.json"}}}})
	if err != nil {
		t.Fatalf("reading the generation: %v", err)
	}
	if asked != "https://elsewhere.example/a.meta.json" {
		t.Errorf("the metadata was read from %q", asked)
	}
}

// No token in the environment is not an error, and the request goes anonymous
// rather than carrying an empty credential. A contributor asking about twelve
// repositories once is inside the anonymous rate and should not have to hold
// one.
func TestNoTokenLeavesTheRequestAnonymous(t *testing.T) {
	t.Setenv(TokenVariable, "   ")

	req, err := http.NewRequest(http.MethodGet, API+"a/b/releases?per_page=100", nil)
	if err != nil {
		t.Fatalf("building the request: %v", err)
	}
	authorise(req)
	if _, carried := req.Header["Authorization"]; carried {
		t.Errorf("the request carries an authorisation header with no token behind it: %q", req.Header.Get("Authorization"))
	}
}

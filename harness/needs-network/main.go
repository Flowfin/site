// The needs-network set: what can only be read from outside, run deliberately.
//
// This program is the one thing in this repository that leaves the machine on
// purpose. It judges a published site rather than this tree, so its verdict
// moves when somebody else's service does, which is exactly why it is not a leg
// of `go run . ci` and not part of `go test ./...`. The gate verb prints on
// every run that this set was not asked for and what asking would cost.
//
// The name comes from the vocabulary the project already publishes rather than
// from here, and where this repository departs from the rule that vocabulary
// belongs to is decisions/0012-the-browser-in-the-gate.md. harness/README.md is
// where both are argued; neither is restated in this file.
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// cannotExist is the path the not-found reading asks for. It is long and it
// says what it is for, because a short one is a path somebody might one day
// create, and the reading would then be about a page rather than about what the
// origin does with a request matching nothing.
const cannotExist = "/this-path-exists-so-that-the-harness-can-read-what-answers-when-nothing-does/"

func main() {
	name := flag.String("name", "flowfin.dev", "the public name to read")
	out := flag.String("out", filepath.Join("harness", "needs-network", "record"),
		"the directory the record is written into")
	reason := flag.String("reason", "asked for by hand",
		"why this run was made, carried into the record")
	timeout := flag.Duration("timeout", 20*time.Second, "how long any one reading may take")
	flag.Parse()

	if err := run(*name, *out, *reason, *timeout, time.Now()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(name, out, reason string, timeout time.Duration, now time.Time) error {
	t := public(name, timeout)
	c := here(now, name, reason)

	readings := []reading{
		readPlainHTTP(t),
		readCertificate(t, c.Taken),
		readPath(t, "the site root", "/"),
		readPath(t, "a path that matches nothing", cannotExist),
		readPath(t, "the catalogue address", "/manifest.json"),
		readPath(t, "the design token file", "/design-tokens.json"),
		skipped("The timing budget on a real network",
			"It needs a browser, which is the needs-browser set rather than this one, and a number "+
				"measured without one would be a number about a transfer rather than about a render."),
		skipped("A keyboard journey with a real assistive technology",
			"No machine decides it. It is run by a person, and a record saying it passed because "+
				"nothing refused it would be the failure this whole directory exists against."),
		skipped("Whether the site serves the bytes this build produced",
			"The comparison is between a digest above and the digest of a file the build wrote, "+
				"and this repository is not the origin yet, so there is nothing on the other side "+
				"of it. decisions/0006-what-answers-at-the-catalogue-address.md is where that day "+
				"is described and docs/domain-cutover.md is the sequence that reaches it."),
	}

	body := render(c, readings)

	if err := os.MkdirAll(out, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.ToSlash(out), err)
	}
	path := filepath.Join(out, c.stamp())
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("writing the record: %w", err)
	}
	fmt.Println(filepath.ToSlash(path))
	return nil
}

// public builds the target for a name served on the public internet. The
// addresses are assembled here and nowhere else, so the suite constructs its
// own target against loopback rather than editing what this one produces.
func public(name string, timeout time.Duration) target {
	return target{
		Name:    name,
		Plain:   "http://" + name + "/",
		Secure:  "https://" + name,
		TLSAddr: net.JoinHostPort(name, "443"),
		Client: &http.Client{
			Timeout: timeout,
			// A redirect is one of the things being read. Following it
			// would replace the answer this set is about with the answer
			// it points at.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		TLS: &tls.Config{ServerName: name, MinVersion: tls.VersionTLS12},
	}
}

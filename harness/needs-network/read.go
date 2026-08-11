// The readings this set takes, and the shape that keeps them honest.
//
// Every reading carries the request that produced it, written as something a
// reader can run, and one of three outcomes: what came back, or the reason
// nothing came back, or that it was not taken at all. The third is the one that
// matters. A set that quietly omitted what it could not do would report a
// shorter list and read as a complete one, and the whole reason this code sits
// outside the gate is that it depends on machines nobody here operates.
//
// Nothing below decides whether the site is good. The readings are observations
// with their commands beside them; the judgements a reader makes from them are
// the reader's, and the record says so.
package main

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// target is where the readings are taken. Every address is a field rather than
// being assembled from the name at each call, so the suite can point the same
// readings at a loopback server. That is what lets this file be covered by
// `go test ./...` while the program it belongs to is the one thing in this
// repository that deliberately leaves the machine.
type target struct {
	// Name is the public name, as an operator would type it.
	Name string
	// Plain is the address reached over http, with no encryption.
	Plain string
	// Secure is the base reached over https, with no trailing slash.
	Secure string
	// TLSAddr is host and port, for the handshake reading.
	TLSAddr string
	// Client makes the requests and follows no redirect: a redirect is one of
	// the things being read rather than something to pass through on the way
	// to a body.
	Client *http.Client
	// TLS is the configuration the handshake reading dials with.
	TLS *tls.Config
}

// reading is one thing that was looked at.
//
// Exactly one of Lines and Failed carries the outcome, and Skipped says the
// reading was never attempted. A reader can tell the three apart without
// knowing what any of them was about, which is the property that stops a run
// that reached nothing from looking like a run that found nothing wrong.
type reading struct {
	// Name is what was read, in the vocabulary of the record rather than of
	// the code.
	Name string
	// Asked is the request, written so a reader can repeat it.
	Asked string
	// Lines is what came back.
	Lines []string
	// Failed is why nothing came back. A reading with this set is never a
	// result.
	Failed string
	// Skipped is why the reading was not attempted at all.
	Skipped string
}

func failed(name, asked string, err error) reading {
	return reading{Name: name, Asked: asked, Failed: err.Error()}
}

// readPlainHTTP asks over http and reports what the answer was rather than
// following it. A name that serves a page here rather than redirecting is
// serving it over a channel anybody on the path can edit, and from a browser
// the two are indistinguishable, because the browser follows the redirect and
// shows a padlock either way.
func readPlainHTTP(t target) reading {
	asked := fmt.Sprintf("GET %s (no redirect followed)", t.Plain)
	resp, err := t.Client.Get(t.Plain)
	if err != nil {
		return failed("plain http", asked, err)
	}
	defer drain(resp)

	lines := []string{fmt.Sprintf("status %s", resp.Status)}
	if loc := resp.Header.Get("Location"); loc != "" {
		lines = append(lines, fmt.Sprintf("location %s", loc))
	} else {
		lines = append(lines, "no location header, so this answer is the page rather than a way to it")
	}
	if hsts := resp.Header.Get("Strict-Transport-Security"); hsts != "" {
		lines = append(lines, fmt.Sprintf("strict-transport-security %s", hsts))
	} else {
		lines = append(lines, "no strict-transport-security header, so nothing but the answer above carries a reader to the secure address")
	}
	return reading{Name: "plain http", Asked: asked, Lines: lines}
}

// readCertificate takes the handshake and reports the leaf it was given. What
// it reports is the certificate's own fields and whether the name is among the
// ones it is for, because "valid" is a word that hides which of several things
// was checked.
func readCertificate(t target, now time.Time) reading {
	asked := fmt.Sprintf("TLS handshake with %s, server name %s", t.TLSAddr, t.Name)
	conn, err := tls.Dial("tcp", t.TLSAddr, t.TLS)
	if err != nil {
		return failed("certificate", asked, err)
	}
	defer conn.Close()

	chain := conn.ConnectionState().PeerCertificates
	if len(chain) == 0 {
		return reading{Name: "certificate", Asked: asked,
			Failed: "the handshake completed and the peer sent no certificate"}
	}
	leaf := chain[0]

	lines := []string{
		fmt.Sprintf("subject %s", leaf.Subject.CommonName),
		fmt.Sprintf("issuer %s", leaf.Issuer.CommonName),
		fmt.Sprintf("names %s", strings.Join(leaf.DNSNames, ", ")),
		fmt.Sprintf("valid from %s until %s",
			leaf.NotBefore.UTC().Format(time.RFC3339), leaf.NotAfter.UTC().Format(time.RFC3339)),
	}
	switch {
	case now.Before(leaf.NotBefore):
		lines = append(lines, "the reading was taken before that window opened")
	case now.After(leaf.NotAfter):
		lines = append(lines, "the reading was taken after that window closed")
	default:
		lines = append(lines, fmt.Sprintf("the reading was taken inside that window, %d day(s) before it closes",
			int(leaf.NotAfter.Sub(now).Hours()/24)))
	}
	if err := leaf.VerifyHostname(t.Name); err != nil {
		lines = append(lines, fmt.Sprintf("the name %s is not one this certificate is for: %v", t.Name, err))
	} else {
		lines = append(lines, fmt.Sprintf("the name %s is one this certificate is for", t.Name))
	}
	return reading{Name: "certificate", Asked: asked, Lines: lines}
}

var titleElement = regexp.MustCompile(`(?is)<title\b[^>]*>(.*?)</title>`)

// readPath fetches one path and reports what came back, with a digest of the
// body. The digest is what makes two readings comparable: whether a published
// site is serving the bytes a build produced is decided by comparing digests,
// and a size and a status code are not enough to tell two pages apart.
func readPath(t target, name, p string) reading {
	url := t.Secure + p
	asked := fmt.Sprintf("GET %s", url)
	resp, err := t.Client.Get(url)
	if err != nil {
		return failed(name, asked, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return failed(name, asked, err)
	}
	sum := sha256.Sum256(body)

	lines := []string{
		fmt.Sprintf("status %s", resp.Status),
		fmt.Sprintf("content-type %s", headerOr(resp, "Content-Type", "absent")),
		fmt.Sprintf("server %s", headerOr(resp, "Server", "absent")),
		fmt.Sprintf("%d byte(s), sha256 %s", len(body), hex.EncodeToString(sum[:])),
	}
	if m := titleElement.FindSubmatch(body); m != nil {
		lines = append(lines, fmt.Sprintf("title %s", collapse(string(m[1]))))
	}
	return reading{Name: name, Asked: asked, Lines: lines}
}

// skipped is a reading that was not taken, carrying why. It is a value rather
// than an omission because the reason is the useful part: a set that lists what
// it did not reach can be argued with, and one that lists only what it reached
// cannot.
func skipped(name, why string) reading {
	return reading{Name: name, Skipped: why}
}

func headerOr(resp *http.Response, name, absent string) string {
	if v := resp.Header.Get(name); v != "" {
		return v
	}
	return absent
}

func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// drain reads and closes a body that is not being reported on, so the
// connection is returned rather than dropped mid-response.
func drain(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

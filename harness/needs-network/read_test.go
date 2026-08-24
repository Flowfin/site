// SPDX-License-Identifier: AGPL-3.0-or-later

// The readings are covered against servers on loopback and nothing else.
//
// The program these tests cover is the one thing in this repository that
// reaches the public network, and the suite that covers it may not, because the
// suite is a leg of the gate. That is not a compromise: what is worth proving
// here is the shape of a reading rather than the state of somebody's site, and
// the shape is what a fixture can hold. Whether the public name answers is what
// a record says, and a record is produced by running the program.
package main

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func loopback(t *testing.T, h http.Handler) target {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return target{
		Name:   "127.0.0.1",
		Plain:  srv.URL + "/",
		Secure: srv.URL,
		Client: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func has(lines []string, want string) bool {
	for _, l := range lines {
		if strings.Contains(l, want) {
			return true
		}
	}
	return false
}

func TestPlainHTTPReportsTheRedirectRatherThanFollowingIt(t *testing.T) {
	tg := loopback(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "https://example.invalid/", http.StatusMovedPermanently)
			return
		}
		t.Errorf("the reading followed the redirect and asked for %s", r.URL.Path)
	}))

	got := readPlainHTTP(tg)
	if got.Failed != "" || got.Skipped != "" {
		t.Fatalf("the reading did not complete: %+v", got)
	}
	if !has(got.Lines, "status 301") {
		t.Errorf("the status is not reported: %q", got.Lines)
	}
	if !has(got.Lines, "location https://example.invalid/") {
		t.Errorf("the redirect target is not reported: %q", got.Lines)
	}
}

func TestPlainHTTPSaysSoWhenNothingCarriesTheReaderOnward(t *testing.T) {
	tg := loopback(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	got := readPlainHTTP(tg)
	if !has(got.Lines, "no location header") {
		t.Errorf("a page served over plain http is not reported as one: %q", got.Lines)
	}
	if !has(got.Lines, "no strict-transport-security header") {
		t.Errorf("the absent transport header is not reported: %q", got.Lines)
	}
}

func TestAReadingThatCouldNotBeTakenIsNotAResult(t *testing.T) {
	tg := loopback(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	tg.Plain = "http://127.0.0.1:1/"

	got := readPlainHTTP(tg)
	if got.Failed == "" {
		t.Fatal("a reading against a port nothing is on reported no failure")
	}
	if len(got.Lines) != 0 {
		t.Errorf("a failed reading carries lines a reader could take for a result: %q", got.Lines)
	}
}

func TestCertificateReportsTheLeafAndWhetherTheNameIsOnIt(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	tg := target{
		Name:    "example.com",
		TLSAddr: u.Host,
		TLS:     &tls.Config{ServerName: "example.com", RootCAs: pool, MinVersion: tls.VersionTLS12},
	}

	got := readCertificate(tg, time.Now())
	if got.Failed != "" {
		t.Fatalf("the handshake did not complete: %s", got.Failed)
	}
	if !has(got.Lines, "names example.com") {
		t.Errorf("the names on the certificate are not reported: %q", got.Lines)
	}
	if !has(got.Lines, "the name example.com is one this certificate is for") {
		t.Errorf("whether the name is covered is not reported: %q", got.Lines)
	}
	if !has(got.Lines, "inside that window") {
		t.Errorf("the validity window is not compared against the moment: %q", got.Lines)
	}
}

func TestCertificateSaysWhenTheNameIsNotOneItIsFor(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	// The handshake is asked for under the name the fixture certificate is
	// for, and the question about coverage is asked about a different one.
	// That separates "the connection worked" from "the certificate is for
	// the name an operator typed", which is the pair a browser collapses
	// into one padlock.
	tg := target{
		Name:    "flowfin.dev",
		TLSAddr: u.Host,
		TLS:     &tls.Config{ServerName: "example.com", RootCAs: pool, MinVersion: tls.VersionTLS12},
	}

	got := readCertificate(tg, time.Now())
	if got.Failed != "" {
		t.Fatalf("the handshake did not complete: %s", got.Failed)
	}
	if !has(got.Lines, "is not one this certificate is for") {
		t.Errorf("a certificate for another name is not reported as one: %q", got.Lines)
	}
}

func TestCertificateReportsAHandshakeThatDidNotHappen(t *testing.T) {
	tg := target{
		Name:    "example.com",
		TLSAddr: "127.0.0.1:1",
		TLS:     &tls.Config{ServerName: "example.com", MinVersion: tls.VersionTLS12},
	}

	got := readCertificate(tg, time.Now())
	if got.Failed == "" {
		t.Fatal("a handshake against a port nothing is on reported no failure")
	}
	if len(got.Lines) != 0 {
		t.Errorf("a failed handshake carries lines a reader could take for a result: %q", got.Lines)
	}
}

func TestPathReportsWhatCameBackAndADigestOfIt(t *testing.T) {
	tg := loopback(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == cannotExist {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("<html><head><title>Site not found</title></head><body></body></html>"))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	got := readPath(tg, "a path that matches nothing", cannotExist)
	if got.Failed != "" {
		t.Fatalf("the reading did not complete: %s", got.Failed)
	}
	if !has(got.Lines, "status 404") {
		t.Errorf("the status is not reported: %q", got.Lines)
	}
	if !has(got.Lines, "title Site not found") {
		t.Errorf("the title is not reported, so two pages of one size cannot be told apart: %q", got.Lines)
	}
	// The digest of exactly those bytes, written out rather than computed
	// here. A test that hashed the fixture again would agree with any
	// digest this code produced, including one taken over the wrong thing.
	const want = "68 byte(s), sha256 6107805060f32f453c05e19c209a9d7b3f818bc71b38fedcf56be75cfe47d378"
	if !has(got.Lines, want) {
		t.Errorf("the digest is not the one those bytes have: %q", got.Lines)
	}
}

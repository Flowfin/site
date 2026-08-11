// The suite over the file a person who found a problem goes looking for.
//
// One case serves what the build wrote and asks for the path over the network,
// because the whole value of this file is that it answers at a fixed address
// and nothing about the bytes on disk says whether it does. The server binds
// loopback and nothing else, and it is the toolchain's own.
//
// No case opens a window, reaches the public network, needs anything outside
// the toolchain or asks for elevation.
package security

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const good = `{"route":"https://example.invalid/report",
	"policy":"https://example.invalid/policy","confirmed":"2026-08-11"}`

// The date is not typed into the file that is produced. Moving the day the
// route was confirmed moves the expiry with it, and nothing else in the output
// changes, which is what makes the field mean the thing it is named after.
func TestTheExpiryIsTheConfirmationDatePlusOneLife(t *testing.T) {
	first, err := Read([]byte(good))
	if err != nil {
		t.Fatalf("reading a contact that breaks nothing: %v", err)
	}
	second, err := Read([]byte(strings.Replace(good, "2026-08-11", "2026-09-20", 1)))
	if err != nil {
		t.Fatalf("reading a contact whose day moved: %v", err)
	}

	before, err := Render(first)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	after, err := Render(second)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}

	if !strings.Contains(before, "Expires: 2027-08-11T00:00:00Z") {
		t.Errorf("the expiry is not the day plus a life:\n%s", before)
	}
	if !strings.Contains(after, "Expires: 2027-09-20T00:00:00Z") {
		t.Errorf("moving the day did not move the expiry:\n%s", after)
	}
	if strings.Replace(after, "2027-09-20", "2027-08-11", 1) != before {
		t.Errorf("moving the day changed something other than the expiry:\n%s\n%s", before, after)
	}
}

// Two renders of one contact are the same bytes, because nothing here reads the
// clock. A build that read one would produce a different site every day and the
// reproducibility check would be the thing that found out.
func TestRenderingTwiceProducesTheSameBytes(t *testing.T) {
	c, err := Read([]byte(good))
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	first, err := Render(c)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	second, err := Render(c)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	if first != second {
		t.Errorf("two renders differ:\n%s\n%s", first, second)
	}
}

// The file names the route and the policy, and it names no language. Which
// language this site publishes in is open on the tracker, and a field here
// would settle it where nobody would look for the answer.
func TestTheFileNamesTheRouteAndThePolicyAndNoLanguage(t *testing.T) {
	c, err := Read([]byte(good))
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	body, err := Render(c)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	for _, want := range []string{
		"Contact: https://example.invalid/report",
		"Policy: https://example.invalid/policy",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the file does not carry %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "Preferred-Languages") {
		t.Errorf("the file answers a question that is open on the tracker:\n%s", body)
	}
}

func TestReadRefusesEachThingItDoesNotUnderstand(t *testing.T) {
	for _, c := range []struct {
		name, body, says string
	}{
		{"not an object", `["a route"]`, "object this file has to be"},
		{"a field nothing reads", strings.Replace(good, `"route"`, `"contact"`, 1), `"contact"`},
		{"an empty route", strings.Replace(good, "https://example.invalid/report", "", 1), "has no route"},
		{"a route that is not a route", strings.Replace(good, "https://example.invalid/report", "write to me", 1), "https address"},
		{"a day that is not a date", strings.Replace(good, "2026-08-11", "August 2026", 1), "not a date"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Read([]byte(c.body))
			if err == nil {
				t.Fatalf("read %s without refusing it", c.name)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal says %q and has to say %q", err, c.says)
			}
		})
	}
}

// The expiry is read back out of what was written rather than recomputed from
// the tree, so a build that wrote the field wrongly does not agree with itself.
func TestExpiryOfReadsWhatWasWrittenAndTellsAFileWithNoneApart(t *testing.T) {
	c, err := Read([]byte(good))
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	body, err := Render(c)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	at, err := ExpiryOf([]byte(body))
	if err != nil {
		t.Fatalf("reading the expiry back: %v", err)
	}
	if at.Format(time.RFC3339) != "2027-08-11T00:00:00Z" {
		t.Errorf("the expiry read back is %s", at.Format(time.RFC3339))
	}

	if _, err := ExpiryOf([]byte("<html lang=\"en\"><title>A page</title></html>")); err != ErrNoExpiry {
		t.Errorf("a file with no expiry answered %v, and it has to be told apart from one written wrongly", err)
	}
	if _, err := ExpiryOf([]byte("Expires: next August\n")); err == nil || err == ErrNoExpiry {
		t.Errorf("an expiry that is not a moment answered %v", err)
	}
}

// The path is the whole point, so this asks for it the way a reporter would,
// from a server that has nothing but what the build wrote.
func TestTheProducedPathAnswersOnAServedBuild(t *testing.T) {
	out := t.TempDir()
	if err := os.MkdirAll(filepath.Join(out, ".well-known"), 0o755); err != nil {
		t.Fatalf("preparing the output: %v", err)
	}
	c, err := Read([]byte(good))
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	body, err := Render(c)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	if err := os.WriteFile(filepath.Join(out, filepath.FromSlash(Path)), []byte(body), 0o644); err != nil {
		t.Fatalf("writing the produced file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(out, "index.html"), []byte("<html lang=\"en\"></html>"), 0o644); err != nil {
		t.Fatalf("writing the landing page: %v", err)
	}

	server := httptest.NewServer(http.FileServer(http.Dir(out)))
	defer server.Close()

	got, code := ask(t, server.URL+"/"+Path)
	if code != http.StatusOK {
		t.Fatalf("the served build answered %d at /%s", code, Path)
	}
	if got != body {
		t.Errorf("what was served is not what the build wrote:\n%s", got)
	}

	// The neighbour: a path nothing wrote does not answer with the file, so
	// the case above is about the path rather than about the server handing
	// out one file for everything.
	if _, code := ask(t, server.URL+"/.well-known/a-path-nothing-wrote"); code == http.StatusOK {
		t.Errorf("a path nothing wrote answered %d", code)
	}
}

func ask(t *testing.T, address string) (string, int) {
	t.Helper()
	resp, err := http.Get(address)
	if err != nil {
		t.Fatalf("asking for %s: %v", address, err)
	}
	defer resp.Body.Close()
	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	return string(body[:n]), resp.StatusCode
}

// The source this repository actually carries. What is asked is that it reads
// and that what it produces has not expired, which is the same question the
// gate asks of the output.
func TestTheContactInThisTreeReadsAndHasNotExpired(t *testing.T) {
	c, err := Load(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("the contact in this tree does not read: %v", err)
	}
	at, err := c.Expires()
	if err != nil {
		t.Fatalf("its expiry does not compute: %v", err)
	}
	if !at.After(time.Now()) {
		t.Fatalf("the contact in this tree expired on %s; %s is where the day it was confirmed is moved forward",
			at.Format(time.RFC3339), File)
	}
}

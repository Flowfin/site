// SPDX-License-Identifier: AGPL-3.0-or-later

// Package security produces the file a person who found a problem in the
// published pages goes looking for.
//
// SECURITY.md is read by somebody who already found the source. The reports
// this project most wants are from people who found a problem in the pages, and
// for them a repository file is nowhere. `.well-known/security.txt` is the one
// path such a person already knows, so the route lives there as well as in the
// tree.
//
// It is produced rather than committed. A committed file describing generated
// output is a second copy that drifts from it, and this one carries a date that
// would go stale in exactly the way a second copy does.
//
// The date is the part of this file that goes wrong by sitting still. An expiry
// that has passed is worse than no file at all, because it tells a finder the
// route was abandoned. So the expiry is not typed: it is a duration added to the
// date the contact was last confirmed, which is data in the tree, and the day
// the sum falls into the past the gate refuses the produced file. That is the
// point of the field. A build that pushed the expiry forward on every run would
// keep the file valid forever without anybody ever having looked at the route
// again.
//
// Nothing here reads the clock. What the build writes is derived from the tree,
// so two builds of one commit produce the same bytes; whether the expiry has
// passed is a question about the world and is asked by the check that reads the
// output rather than by the build that wrote it.
package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// File is what the build reads, and Path is where what it produces lands in the
// output.
const (
	File = "data/security-contact.json"
	Path = ".well-known/security.txt"
)

// Life is how long a confirmation is good for. A year is the longest the format
// allows and it is taken whole: the field exists to make somebody look at the
// route again, and a shorter life would make the gate ask more often than there
// is anything new to say.
const Life = 365 * 24 * time.Hour

// DateFormat is how the confirmation date is written in the tree. A date and not
// a moment, because what is being recorded is the day somebody checked.
const DateFormat = "2006-01-02"

// Contact is what the tree says about where a report goes.
type Contact struct {
	// Route is the address a report is made at.
	Route string `json:"route"`
	// Policy is where what follows a report is written down.
	Policy string `json:"policy"`
	// Confirmed is the day the route above was last checked to be the one a
	// report should go to. Moving it forward is the act the expiry exists to
	// force, and it is a change somebody makes rather than one a build makes.
	Confirmed string `json:"confirmed"`
}

var fieldNames = map[string]bool{"route": true, "policy": true, "confirmed": true}

// Load reads what the tree says. Every refusal names the field it is about,
// because a message saying only that the file is invalid makes the next person
// read it line by line.
func Load(root string) (Contact, error) {
	name := filepath.Join(root, filepath.FromSlash(File))
	body, err := os.ReadFile(filepath.Clean(name))
	if err != nil {
		return Contact{}, fmt.Errorf("reading %s: %w", File, err)
	}
	c, err := Read(body)
	if err != nil {
		return Contact{}, fmt.Errorf("%s: %w", File, err)
	}
	return c, nil
}

// Read parses and refuses everything it does not understand.
func Read(body []byte) (Contact, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return Contact{}, fmt.Errorf("is not the object this file has to be: %w", err)
	}
	for field := range raw {
		if !fieldNames[field] {
			return Contact{}, fmt.Errorf("carries the field %q, which is not a field of this file", field)
		}
	}
	var c Contact
	if err := json.Unmarshal(body, &c); err != nil {
		return Contact{}, fmt.Errorf("does not carry the fields this file has to carry: %w", err)
	}
	for field, value := range map[string]string{"route": c.Route, "policy": c.Policy, "confirmed": c.Confirmed} {
		if strings.TrimSpace(value) == "" {
			return Contact{}, fmt.Errorf("has no %s, and every field is required", field)
		}
	}
	for field, value := range map[string]string{"route": c.Route, "policy": c.Policy} {
		if !strings.HasPrefix(value, "https://") {
			return Contact{}, fmt.Errorf("gives %q for the %s, and a route a reporter can use is an https address", value, field)
		}
	}
	if _, err := c.Expires(); err != nil {
		return Contact{}, err
	}
	return c, nil
}

// Expires is the day the file stops being trustworthy, which is the day the
// route was last confirmed plus one life.
func (c Contact) Expires() (time.Time, error) {
	day, err := time.Parse(DateFormat, c.Confirmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("gives %q for the day the route was confirmed, and that is not a date written as %s", c.Confirmed, DateFormat)
	}
	return day.UTC().Add(Life), nil
}

// Render writes the file. The fields are the ones the format defines and the
// comment says why the file is there; what the route is and what state it is in
// is the policy rather than a sentence here, because a state repeated in two
// places is a state that disagrees with itself.
//
// It carries no preferred language. Which language this site publishes in is
// open on the tracker, and a field answering it here would be that question
// settled in the one place nobody would look for the answer.
func Render(c Contact) (string, error) {
	expires, err := c.Expires()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("# A problem found in these pages goes to the route below rather than to the\n")
	b.WriteString("# public tracker, where a report is readable by everybody from the moment it\n")
	b.WriteString("# is made. What that route is and what state it is in is the policy.\n")
	fmt.Fprintf(&b, "Contact: %s\n", c.Route)
	fmt.Fprintf(&b, "Policy: %s\n", c.Policy)
	fmt.Fprintf(&b, "Expires: %s\n", expires.Format(time.RFC3339))
	return b.String(), nil
}

// ErrNoExpiry is what reading a file that carries no expiry answers with. It is
// separate because the check that reads the output has to tell a file that is
// not this one from this one written wrongly.
var ErrNoExpiry = errors.New("carries no expiry")

// ExpiryOf reads the expiry back out of a produced file. The check reads what
// was written rather than recomputing it from the tree: a build that wrote the
// field wrongly would agree with itself perfectly.
func ExpiryOf(body []byte) (time.Time, error) {
	for _, line := range strings.Split(string(body), "\n") {
		rest, ok := strings.CutPrefix(line, "Expires:")
		if !ok {
			continue
		}
		at, err := time.Parse(time.RFC3339, strings.TrimSpace(rest))
		if err != nil {
			return time.Time{}, fmt.Errorf("carries the expiry %q, which is not a moment written as %s", strings.TrimSpace(rest), time.RFC3339)
		}
		return at, nil
	}
	return time.Time{}, ErrNoExpiry
}

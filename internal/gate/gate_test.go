// What the gate says about itself.
//
// The legs are supplied by each test rather than taken from the tree, because a
// test that ran the real set would run `go test ./...` from inside `go test`
// and call itself. What is proved here is the reporting: a run that covers less
// than everything has to say so, in its own output, whether it passed or
// stopped at the first failure.
package gate

import (
	"errors"
	"strings"
	"testing"
)

func TestARunNamesTheSetItDidNotAskFor(t *testing.T) {
	var log strings.Builder
	if err := run(".", &log, nil); err != nil {
		t.Fatalf("a run with no leg refused the tree: %v", err)
	}
	got := log.String()

	for _, s := range outside() {
		if !strings.Contains(got, s.name+" was not asked for") {
			t.Errorf("the run does not say that %s was not asked for:\n%s", s.name, got)
		}
		if !strings.Contains(got, s.cost) {
			t.Errorf("the run does not say what asking for %s would cost:\n%s", s.name, got)
		}
		if !strings.Contains(got, s.how) {
			t.Errorf("the run does not say how to ask for %s:\n%s", s.name, got)
		}
	}
}

func TestTheSetIsNamedEvenWhenTheRunStopsAtItsFirstLeg(t *testing.T) {
	// The sentence is printed before the legs for this reason. A run that
	// only named what it left out on the way to a green result would say
	// nothing on exactly the runs somebody reads most carefully.
	var log strings.Builder
	err := run(".", &log, []leg{
		{"refuses", func(string) (string, error) { return "", errors.New("this leg refuses everything") }},
		{"never reached", func(string) (string, error) { return "ok", nil }},
	})
	if err == nil {
		t.Fatal("a run whose first leg failed reported no error")
	}
	got := log.String()

	if !strings.Contains(got, "needs-network was not asked for") {
		t.Errorf("a failing run does not name the set it did not ask for:\n%s", got)
	}
	if !strings.Contains(got, "Not reached: never reached.") {
		t.Errorf("a failing run does not name the legs it never reached:\n%s", got)
	}
}

func TestEveryOutsideSetCarriesACostAndAWayToRunIt(t *testing.T) {
	// A set named with no cost beside it is an invitation rather than a
	// disclosure, and one with no command is a reader being told to go and
	// find it.
	for _, s := range outside() {
		if s.name == "" || s.cost == "" || s.how == "" {
			t.Errorf("an outside set is incomplete: %+v", s)
		}
	}
}

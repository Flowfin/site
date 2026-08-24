// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"strings"
	"testing"
	"time"
)

func fixedConditions() conditions {
	return conditions{
		Taken:  time.Date(2026, 8, 11, 21, 15, 0, 0, time.UTC),
		Name:   "flowfin.dev",
		Go:     "go1.26.5",
		OS:     "windows/amd64",
		Reason: "asked for by hand",
	}
}

func TestTheStampCarriesNoColon(t *testing.T) {
	// A colon is a path separator on one of the systems this repository is
	// developed on, so a record named with one is a run that fails at the
	// last line after every reading has been taken.
	if got := fixedConditions().stamp(); strings.Contains(got, ":") {
		t.Errorf("the record name %q cannot be written on every system this is run on", got)
	}
}

func TestTheRecordSeparatesWhatWasReadFromWhatFailedAndWhatWasNotAsked(t *testing.T) {
	got := render(fixedConditions(), []reading{
		{Name: "plain http", Asked: "GET http://flowfin.dev/", Lines: []string{"status 301 Moved Permanently"}},
		{Name: "certificate", Asked: "TLS handshake", Failed: "dial tcp: connection refused"},
		skipped("a keyboard journey", "no machine decides it"),
	})

	if !strings.Contains(got, "1 reading(s) taken, 1 that could not be taken, 1 not asked for.") {
		t.Errorf("the three outcomes are not counted separately:\n%s", got)
	}
	read := section(got, "## What was read")
	if strings.Contains(read, "certificate") {
		t.Errorf("a failed reading appears among the results:\n%s", read)
	}
	if strings.Contains(read, "keyboard") {
		t.Errorf("a reading nobody asked for appears among the results:\n%s", read)
	}
	if !strings.Contains(section(got, "## What could not be read"), "connection refused") {
		t.Errorf("the reason a reading failed is not carried:\n%s", got)
	}
	if !strings.Contains(section(got, "## What this run did not ask for"), "no machine decides it") {
		t.Errorf("the reason a reading was not taken is not carried:\n%s", got)
	}
}

func TestTheRecordCarriesTheMomentAndTheMachine(t *testing.T) {
	got := render(fixedConditions(), nil)

	for _, want := range []string{
		"2026-08-11T21:15:00Z",
		"name       flowfin.dev",
		"toolchain  go1.26.5",
		"machine    windows/amd64",
		"run        asked for by hand",
		"from one machine on one network, at one moment",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the record does not carry %q:\n%s", want, got)
		}
	}
}

func TestARunThatReadNothingSaysSoRatherThanPrintingAnEmptySection(t *testing.T) {
	got := render(fixedConditions(), []reading{
		{Name: "plain http", Asked: "GET http://flowfin.dev/", Failed: "dial tcp: connection refused"},
	})

	if !strings.Contains(got, "0 reading(s) taken, 1 that could not be taken, 0 not asked for.") {
		t.Errorf("a run that read nothing is not counted as one:\n%s", got)
	}
	if !strings.Contains(section(got, "## What was read"), "Nothing.") {
		t.Errorf("a run that read nothing leaves a section a reader would take for a result:\n%s", got)
	}
}

// section returns the text under one heading, so a test can ask where a line
// appeared rather than only whether it appeared somewhere.
func section(body, heading string) string {
	i := strings.Index(body, heading)
	if i < 0 {
		return ""
	}
	rest := body[i+len(heading):]
	if j := strings.Index(rest, "\n## "); j >= 0 {
		return rest[:j]
	}
	return rest
}

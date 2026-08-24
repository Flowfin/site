// SPDX-License-Identifier: AGPL-3.0-or-later

// What the landing page says about the clients, and why it is a value rather
// than a paragraph.
//
// The first thing this project says about itself is about clients, and no
// client is released. A reader who arrives because of that sentence has nothing
// to download and, until now, nothing on the page telling them so. The repair
// is not a better paragraph. A claim about whether something is available that
// lives in prose is a claim somebody has to remember to change, and the day it
// stops being true is the day nobody is reading that paragraph.
//
// So the claim is a value in the tree and the sentence is composed from it.
// Changing what the page says is changing the value, which is one line in one
// file, and there is no wording anywhere that can go on saying the old thing
// after the value has moved.
//
// The vocabulary is deliberately not the roster's. The three state words there
// say something about a plugin repository and are computed from what that
// repository has published; a client is not a roster row today and borrowing a
// word that means something else would be the drift this milestone exists
// against. What replaces this file is the day a client becomes something a
// person can install: it stops being a sentence here and becomes a row, with
// its kind named, in the file the pages are generated from.
package site

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ClientsFile is what the claim is read from.
const ClientsFile = "data/clients.json"

// The two things the file may say, and there is no third. A closed set is what
// makes the sentence composable at all: a free-text claim would be prose again,
// one file further down.
const (
	// NoneReleased is that nothing a person can install exists yet.
	NoneReleased = "none-released"
	// Released is that at least one does, and then the file has to say
	// where a reader gets it.
	Released = "released"
)

// clients is what the tree says. Intent is what the clients are meant to be,
// in the words a reader reads, and it is data rather than template text for the
// same reason the availability is: a description of software that does not
// exist yet is the sentence most likely to go quietly out of date.
type clients struct {
	Intent       string `json:"intent"`
	Availability string `json:"availability"`
	// Where a reader gets one. It carries something only when something is
	// released, because a route to software nobody has published is an
	// address that answers with nothing.
	Where string `json:"where"`
}

var clientFields = map[string]bool{"intent": true, "availability": true, "where": true}

// readClients reads the claim and refuses everything it does not understand.
// Every refusal is a shape that renders as a finished sentence, which is the
// class of mistake this file exists to catch rather than a syntax error.
func readClients(name string) (clients, error) {
	body, err := os.ReadFile(filepath.Clean(name))
	if err != nil {
		return clients{}, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return clients{}, fmt.Errorf("%s is not the object this file has to be: %w", ClientsFile, err)
	}
	for field := range raw {
		if !clientFields[field] {
			return clients{}, fmt.Errorf("%s carries the field %q, which is not a field of this file", ClientsFile, field)
		}
	}
	var c clients
	if err := json.Unmarshal(body, &c); err != nil {
		return clients{}, fmt.Errorf("%s does not carry the fields this file has to carry: %w", ClientsFile, err)
	}

	if strings.TrimSpace(c.Intent) == "" {
		return clients{}, fmt.Errorf(
			"%s says nothing about what the clients are, and a page that leads with them and describes none of them is the page this file exists to prevent",
			ClientsFile)
	}
	switch c.Availability {
	case NoneReleased:
		if strings.TrimSpace(c.Where) != "" {
			return clients{}, fmt.Errorf(
				"%s says %s and still names %q as where to get one, which no reader will see and which whoever wrote it has stopped looking at",
				ClientsFile, NoneReleased, c.Where)
		}
	case Released:
		if strings.TrimSpace(c.Where) == "" {
			return clients{}, fmt.Errorf(
				"%s says %s and names nowhere to get one, which is a page telling a reader something exists and leaving them to look for it",
				ClientsFile, Released)
		}
	case "":
		return clients{}, fmt.Errorf(
			"%s declares no availability, and %s and %s are the only two this file may declare",
			ClientsFile, NoneReleased, Released)
	default:
		return clients{}, fmt.Errorf(
			"%s declares the availability %q, and %s and %s are the only two this file may declare",
			ClientsFile, c.Availability, NoneReleased, Released)
	}
	return c, nil
}

// sentence is what the page says, composed from the value. Both halves come out
// of the file, so there is no wording here that survives the value changing
// underneath it.
func (c clients) sentence() string {
	if c.Availability == Released {
		return fmt.Sprintf(
			"The clients this project means to build are %s. At least one of them is released, and %s is where a reader gets it.",
			c.Intent, c.Where)
	}
	return fmt.Sprintf(
		"The clients this project means to build are %s. None of them is released, so there is nothing here to download and no page describing one as though there were.",
		c.Intent)
}

// sayWhatTheClientsAre puts the sentence on the page, directly after the
// paragraph that says what this project is. That position is the point: a
// reader who reads far enough to learn that clients are part of the plan has
// already read the line telling them none exists, rather than finding it after
// going to look for a download.
func sayWhatTheClientsAre(paragraphs []string, said string) []string {
	if len(paragraphs) == 0 {
		return []string{said}
	}
	out := make([]string, 0, len(paragraphs)+1)
	out = append(out, paragraphs[0], said)
	return append(out, paragraphs[1:]...)
}

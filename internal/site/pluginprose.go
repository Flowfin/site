// SPDX-License-Identifier: AGPL-3.0-or-later

// The paragraphs a plugin page carries beyond the sentence the roster holds.
//
// decisions/0010-where-the-plugin-page-prose-comes-from.md puts them in this
// repository, one file per roster identifier, and keeps the roster at its one
// sentence. The identifier is the only join: the build takes the file whose
// name matches the row it is rendering and nothing else, so a thirteenth plugin
// is a row and a file, and neither this file nor the frame nor any check learns
// a number when it arrives.
//
// Both halves of that join are refused. A row whose file is missing is a plugin
// the site is silently thin about; a file matching no row is a page nobody can
// reach and nobody will notice has gone stale. The two have opposite causes and
// the same repair, and each refusal names the identifier rather than the path,
// because the identifier is the thing the two sides share and the path is a
// consequence of it.
//
// Nothing here is fetched. The prose is read out of the tree, which is what
// keeps the build reproducible and offline, and it reaches the page through the
// same engine every value out of the roster does, so a paragraph carrying a
// bracket renders as text rather than as markup.
package site

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// PluginProseDir is where the per-plugin prose lives. It is a container inside
// the content directory rather than the content directory itself, because the
// files beside it are the pages this site writes by name and a roster
// identifier is free to be any of those words. A flat directory would make the
// two populations one, and the walk that refuses a file matching no row could
// not tell a plugin nobody carries from the landing page.
const PluginProseDir = ContentDir + "/" + PluginsDir

// proseSuffix is what a prose file is named with, matching the other files the
// content directory holds.
const proseSuffix = ".txt"

// readPluginProse reads one file per row and refuses every way the two halves
// can disagree, in one pass.
//
// It answers with every reason rather than the first, because the two failures
// arrive together in practice: a plugin renamed in the roster is at once a row
// with no file and a file with no row, and a reader shown only the first half
// repairs one side and meets the other.
func readPluginProse(root string, rows []plugin) (map[string][]string, error) {
	held, err := proseFilesInTheTree(root)
	if err != nil {
		return nil, err
	}

	var reasons []string
	prose := make(map[string][]string, len(rows))
	carried := make(map[string]bool, len(rows))
	for _, r := range rows {
		carried[r.ID] = true
		name, ok := held[r.ID]
		if !ok {
			reasons = append(reasons, fmt.Sprintf(
				"the roster carries the row %s and %s holds no prose for it, so that plugin's page would be the table row with more space around it",
				r.ID, PluginProseDir))
			continue
		}
		paragraphs, reason := proseIn(name, r.ID)
		if reason != "" {
			reasons = append(reasons, reason)
			continue
		}
		prose[r.ID] = paragraphs
	}

	for _, id := range identifiersIn(held) {
		if !carried[id] {
			reasons = append(reasons, fmt.Sprintf(
				"%s holds prose for %s and the roster carries no row with that identifier, so it is a page nobody can reach and nobody will notice has gone stale",
				PluginProseDir, id))
		}
	}

	if len(reasons) > 0 {
		return nil, fmt.Errorf("the roster and %s disagree, %d reason(s):\n  %s",
			PluginProseDir, len(reasons), strings.Join(reasons, "\n  "))
	}
	return prose, nil
}

// proseFilesInTheTree answers with the prose files, keyed by the identifier each
// one is named for.
//
// An absent directory is an empty set rather than an error. The refusal a tree
// with no prose at all deserves is one per row naming the plugin it is thin
// about, which the caller produces, and an error here would replace twelve
// identifiers with one path.
func proseFilesInTheTree(root string) (map[string]string, error) {
	dir := filepath.Join(root, filepath.FromSlash(PluginProseDir))
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", PluginProseDir, err)
	}

	held := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), proseSuffix) {
			return nil, fmt.Errorf(
				"%s is read as one prose file per roster identifier and %s is not one, so it is bytes in the tree that no page renders and no row is missing for",
				PluginProseDir, path.Join(PluginProseDir, e.Name()))
		}
		held[strings.TrimSuffix(e.Name(), proseSuffix)] = filepath.Join(dir, e.Name())
	}
	return held, nil
}

// proseIn reads one file into its paragraphs, or the reason it was refused.
//
// A block opening like a keyword is refused for the reason the pages give it: a
// misspelled keyword read as prose empties whatever it was for and the rendered
// page looks finished either way. These files carry no keyword at all, so any
// such block is a mistake rather than a wrong one.
func proseIn(name, id string) (paragraphs []string, reason string) {
	read, err := blocks(name)
	if err != nil {
		return nil, fmt.Sprintf("the prose for %s could not be read: %v", id, err)
	}
	for _, b := range read {
		joined := strings.Join(b, " ")
		if keywordLine.MatchString(joined) {
			return nil, fmt.Sprintf(
				"the prose for %s opens a block %q, and these files carry no keyword, so a block that opens like one is read as a paragraph and whatever it was for is lost: %s",
				id, strings.SplitN(joined, ":", 2)[0]+":", short(joined))
		}
		paragraphs = append(paragraphs, joined)
	}
	if len(paragraphs) == 0 {
		return nil, fmt.Sprintf(
			"the prose for %s carries no paragraph, and a file that exists and says nothing passes the same walk a written one does",
			id)
	}
	return paragraphs, ""
}

// identifiersIn answers with the identifiers held, in a fixed order, so a tree with several
// unmatched files is refused with the same reasons in the same order on every
// run and a reader comparing two runs is comparing the trees.
func identifiersIn(held map[string]string) []string {
	ids := make([]string, 0, len(held))
	for id := range held {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

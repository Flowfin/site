// The privacy page is prose with a register attached to every statement on it,
// and this file is what reads that.
//
// A page saying a site does not do something is worth what a reader can check,
// and a reader can check none of it from the outside. So each statement carries
// what stands behind it: the name of the invariant that refuses a page breaking
// it, the issue that would refuse it where nothing does yet, or nothing at all
// where the statement is about what is true anyway and is not a promise. The
// three registers are the page. A file that collapsed them would be a page
// asking to be believed.
//
// It fails closed on every shape that moves a statement between registers,
// because that is the failure this file exists to prevent and it is the one
// failure that is invisible in the rendered page. A mistyped keyword becomes a
// paragraph and takes a claim out of the register anything reads. A statement
// with nothing named behind it renders as though something refuses it. An issue
// number where a check name goes reads as a property and is a promise.
//
// What is not decided here is whether a named check exists, because this
// package cannot ask: the rules live in the package that reads the output this
// one writes, and the import would run the other way. That half is an invariant
// over the produced page, which is also where it belongs, since a page carrying
// a name nothing answers to is a defect of the page rather than of the file.
package site

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// PrivacyFile is the prose and PrivacyPath is where the produced page lands.
// The address is the one decisions/0008-the-url-shape.md gives it, and the file
// name inside it is what a static host serves for a directory address without
// being told to.
const (
	PrivacyFile = ContentDir + "/privacy.txt"
	PrivacyPath = "privacy/index.html"
)

// The keyword a statement opens with. They are three because the page is three
// registers, and the parser refuses a fourth rather than guessing which of the
// three somebody meant.
const (
	checkedKeyword  = "checked:"
	promisedKeyword = "promised:"
	residualKeyword = "residual:"
)

// marker is what a statement carries at its end, in square brackets. One
// expression reads both kinds because what separates them is what is inside,
// and reading them apart would let a statement carrying neither shape through
// as a statement carrying nothing.
var marker = regexp.MustCompile(`^(.*?)\s*\[([^\]]*)\]$`)

// writePrivacy renders the privacy page out of the prose and into the output,
// through the same template every other page goes through, so a statement on it
// is escaped by the path the rest of the site is escaped by.
//
// An absent source is reported rather than passed over, the way the assets walk
// and the reporting route report an absent one. A run that produced no privacy
// page must not read like a run that had nothing to produce. A source that is
// there and cannot be read is a refusal, because a privacy page rendered from
// half a file is the one page where a missing statement reads as an absence of
// the thing it was about.
func writePrivacy(root, out, label string, tmpl *template.Template, said descriptions, log io.Writer) ([]string, error) {
	source := filepath.Join(root, filepath.FromSlash(PrivacyFile))
	if _, err := os.Stat(source); os.IsNotExist(err) {
		fmt.Fprintf(log, "no %s in the tree, so no privacy page was written\n", PrivacyFile)
		return nil, nil
	}
	p, err := readPrivacy(source)
	if err != nil {
		return nil, fmt.Errorf("reading the privacy prose: %w", err)
	}
	p.locate(PrivacyPath)
	if err := said.add(PrivacyPath, p.Description); err != nil {
		return nil, err
	}
	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, p); err != nil {
		return nil, fmt.Errorf("rendering the privacy page: %w", err)
	}
	name := filepath.Join(out, filepath.FromSlash(PrivacyPath))
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return nil, fmt.Errorf("creating the directory for %s: %w", PrivacyPath, err)
	}
	if err := os.WriteFile(name, []byte(rendered.String()), 0o644); err != nil {
		return nil, fmt.Errorf("writing %s: %w", PrivacyPath, err)
	}
	slashed := path.Join(label, PrivacyPath)
	fmt.Fprintf(log, "wrote %s (%d bytes, %d checked, %d promised, %d residual)\n",
		slashed, rendered.Len(), len(p.Claims), len(p.Promises), len(p.Residuals))
	return []string{slashed}, nil
}

// readPrivacy reads the prose and returns the page it makes, or every reason it
// will not. Every reason is collected rather than the first, because a file with
// three mistakes in it is three repairs and reporting one of them costs three
// runs.
func readPrivacy(name string) (page, error) {
	read, err := blocks(name)
	if err != nil {
		return page{}, err
	}

	p, reasons := readBlocks(read)
	if len(reasons) > 0 {
		return page{}, fmt.Errorf("%s was refused, %d reason(s):\n  %s",
			filepath.ToSlash(name), len(reasons), strings.Join(reasons, "\n  "))
	}
	return p, nil
}

// readBlocks turns the blocks into the page. A block is one statement or one
// paragraph, and which it is decided by how it opens.
func readBlocks(blocks [][]string) (page, []string) {
	var p page
	var reasons []string

	for i, b := range blocks {
		joined := strings.Join(b, " ")
		switch {
		case i == 0:
			p.Title = joined
		case strings.HasPrefix(joined, descriptionKeyword):
			text, reason := describe(joined)
			if reason != "" {
				reasons = append(reasons, reason)
				continue
			}
			p.Description = text
		case strings.HasPrefix(joined, checkedKeyword):
			text, name, err := split(joined, checkedKeyword)
			switch {
			case err != "":
				reasons = append(reasons, err)
			case strings.HasPrefix(name, "#"):
				reasons = append(reasons, fmt.Sprintf(
					"a checked statement names %q, which is an issue rather than a check, and an issue is what a promise names: %s", name, short(text)))
			default:
				p.Claims = append(p.Claims, claim{Text: text, RefusedBy: name})
			}
		case strings.HasPrefix(joined, promisedKeyword):
			text, name, err := split(joined, promisedKeyword)
			switch {
			case err != "":
				reasons = append(reasons, err)
			case !strings.HasPrefix(name, "#"):
				reasons = append(reasons, fmt.Sprintf(
					"a promised statement names %q, and a promise names the issue that would refuse it rather than a check, because naming a check is what says one exists: %s", name, short(text)))
			default:
				p.Promises = append(p.Promises, promise{Text: text, Waiting: name})
			}
		case strings.HasPrefix(joined, residualKeyword):
			text := strings.TrimSpace(strings.TrimPrefix(joined, residualKeyword))
			if m := marker.FindStringSubmatch(text); m != nil {
				reasons = append(reasons, fmt.Sprintf(
					"a residual statement names %q, and a residual is what nothing refuses and nothing promises, so naming anything behind it is a claim in the wrong register: %s", m[2], short(text)))
				continue
			}
			if text == "" {
				reasons = append(reasons, "a residual statement carries no sentence")
				continue
			}
			p.Residuals = append(p.Residuals, text)
		case keywordLine.MatchString(joined):
			reasons = append(reasons, fmt.Sprintf(
				"a block opens %q, which is not %s and not one of the three registers %s, %s and %s, and a block that opens like a statement and is read as a paragraph loses whatever stood behind it: %s",
				strings.SplitN(joined, ":", 2)[0]+":", descriptionKeyword, checkedKeyword, promisedKeyword, residualKeyword, short(joined)))
		default:
			p.Paragraphs = append(p.Paragraphs, joined)
		}
	}

	if p.Title == "" {
		reasons = append(reasons, "the file carries no title line")
	}
	if p.Description == "" {
		reasons = append(reasons, missingDescription())
	}
	if len(p.Claims) == 0 {
		reasons = append(reasons, "the file carries no checked statement, and a privacy page with nothing checked on it is the promise this page exists in order not to be")
	}
	if len(p.Residuals) == 0 {
		reasons = append(reasons, "the file carries no residual statement, and what a host sees is true whatever this site does, so a page that leaves it out is a page whose other statements read wider than they are")
	}
	return p, reasons
}

// split takes the keyword off a statement and reads the marker at its end. It
// returns the sentence, what the marker named, and the reason the statement was
// refused where it was.
func split(joined, keyword string) (text, name, reason string) {
	rest := strings.TrimSpace(strings.TrimPrefix(joined, keyword))
	m := marker.FindStringSubmatch(rest)
	if m == nil {
		return "", "", fmt.Sprintf(
			"a %s statement names nothing behind it, and a statement with nothing named behind it renders exactly like one that is refused: %s",
			strings.TrimSuffix(keyword, ":"), short(rest))
	}
	text, name = strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
	switch {
	case text == "":
		return "", "", fmt.Sprintf("a %s statement carries a name and no sentence", strings.TrimSuffix(keyword, ":"))
	case name == "":
		return "", "", fmt.Sprintf(
			"a %s statement carries empty brackets, which reads on the page as a name and is not one: %s",
			strings.TrimSuffix(keyword, ":"), short(text))
	}
	return text, name, ""
}

// short is how a statement appears inside a refusal. The whole sentence would
// bury the reason it is being refused, and the opening of it is enough to find
// the line in the file.
func short(s string) string {
	const most = 60
	if len(s) <= most {
		return s
	}
	return s[:most] + "..."
}

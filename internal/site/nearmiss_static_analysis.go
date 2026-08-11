// This file exists for exactly one run and the commit after it removes it. It
// carries one instance of each shape the rule set in tools/semgrep/rules.yml
// refuses, so that the rules are shown to bite on the server against a real
// checkout rather than asserted to on somebody's machine.
//
// Nothing calls it and nothing may. It compiles, it formats, and the suite and
// the build pass over it, which is the point: every other check stays green so
// that the run below it says which check refused and for what.
package site

import (
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	texttemplate "text/template"

	"github.com/Flowfin/site/internal/roster"
)

func nearMissEveryRefusedShape(e roster.Entry, root, name string) ([]byte, error) {
	_ = template.HTML(e.Summary)
	_, _ = texttemplate.New("page").Parse("{{.}}")
	_ = exec.Command("go", "version")
	pagePath := filepath.Join(root, e.ID+".html")
	_ = pagePath
	return os.ReadFile(name)
}

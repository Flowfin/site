// Package nearmiss carries one deliberate defect, for one run, so that what the
// analysis does with it can be read rather than assumed. A scan that has never
// refused anything is a scan nobody has tested. It is removed from this branch
// before the branch merges, and nothing calls into it.
package nearmiss

import (
	"net/http"
	"os/exec"
)

// Handle builds a shell command out of a query parameter, which is the shape the
// analysis is expected to name. Nothing reaches it: no route is registered
// anywhere in this tree and this site serves static files.
func Handle(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	out, err := exec.Command("sh", "-c", "echo "+name).CombinedOutput()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(out); err != nil {
		return
	}
}
